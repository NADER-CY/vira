package main

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var sixDigits = regexp.MustCompile(`[0-9]{6}`)

type appUser struct{ id, token string }

func bearer(u appUser) map[string]string {
	return map[string]string{"Authorization": "Bearer " + u.token}
}

// signUp registers a user through the real OTP flow: the code is read from the SMS simulator inbox.
func signUp(api, devsim, phone string) (appUser, bool) {
	code, ch, raw := call("POST", api+"/v1/auth/otp", map[string]string{"phone": phone}, map[string]string{"X-Device-Id": "smoke-" + phone})
	if code != http.StatusAccepted {
		check("OTP requested for "+phone, false, fmt.Sprintf("%d %s", code, raw))
		return appUser{}, false
	}
	var otp string
	for i := 0; i < 20 && otp == ""; i++ {
		_, inbox, _ := call("GET", devsim+"/sms/v1/messages", nil, nil)
		items, _ := inbox["data"].([]any)
		for _, it := range items {
			m := it.(map[string]any)
			if m["to"] == phone {
				otp = sixDigits.FindString(fmt.Sprint(m["body"]))
				break
			}
		}
		if otp == "" {
			time.Sleep(200 * time.Millisecond)
		}
	}
	check("6-digit OTP delivered to the SMS simulator for "+phone, len(otp) == 6, "no code")
	code, in, raw := call("POST", api+"/v1/auth/otp/verify", map[string]any{"challenge_id": ch["challenge_id"], "code": otp,
		"device":   map[string]any{"id": "smoke-device-" + phone, "platform": "android"},
		"consents": map[string]any{"terms": true, "privacy": true, "age_confirmation": true, "text_version": "smoke"}}, nil)
	check("sign-up with OTP for "+phone, code == 200 && in["access_token"] != nil, fmt.Sprintf("%d %s", code, raw))
	if code != 200 {
		return appUser{}, false
	}
	return appUser{id: fmt.Sprint(in["user_id"]), token: fmt.Sprint(in["access_token"])}, true
}

// phase2 exercises identity, social and moderation through the running stack; returns the reported post id.
func phase2(api, devsim string) (string, bool) {
	suffix := fmt.Sprintf("%07d", time.Now().UnixNano()%10_000_000)
	a, ok1 := signUp(api, devsim, "0912"+suffix)
	b, ok2 := signUp(api, devsim, "0913"+suffix)
	if !ok1 || !ok2 {
		return "", false
	}
	code, prof, raw := call("GET", api+"/v1/me/profile", nil, bearer(a))
	check("own profile", code == 200 && prof["handle"] != nil, raw)
	code, post, raw := call("POST", api+"/v1/posts", map[string]any{"caption": "پست آزمایشی smoke", "visibility": "public"}, bearer(a))
	check("create post", code == 201, fmt.Sprintf("%d %s", code, raw))
	pid := fmt.Sprint(post["id"])
	code, fr, raw := call("POST", api+"/v1/users/"+a.id+"/follow", nil, bearer(b))
	check("follow", code == 200 && fr["state"] == "following", raw)
	code, feed, raw := call("GET", api+"/v1/feed/following", nil, bearer(b))
	items, _ := feed["data"].([]any)
	check("following feed shows the post", code == 200 && len(items) == 1, raw)
	code, like, raw := call("PUT", api+"/v1/posts/"+pid+"/like", nil, bearer(b))
	check("like", code == 200 && like["like_count"] == float64(1), raw)
	code, _, raw = call("POST", api+"/v1/posts/"+pid+"/comments", map[string]any{"body": "نظر آزمایشی"}, bearer(b))
	check("comment", code == 201, raw)
	code, _, raw = call("POST", api+"/v1/reports", map[string]any{"target_type": "post", "target_id": pid, "reason": "spam"}, bearer(b))
	check("report post", code == 201, raw)
	code, _, raw = call("POST", api+"/v1/auth/logout", nil, bearer(b))
	check("logout", code == 204, raw)
	code, _, _ = call("GET", api+"/v1/me/account", nil, bearer(b))
	check("token refused after logout", code == 401, fmt.Sprint(code))
	return pid, true
}

// phase2Admin reviews the report and checks user management with the signed-in admin client.
func phase2Admin(admin, pid string) {
	code, users, raw := call("GET", admin+"/v1/admin/users?limit=5", nil, nil)
	list, _ := users["data"].([]any)
	check("admin user list", code == 200 && len(list) >= 2, raw)
	code, reps, raw := call("GET", admin+"/v1/admin/reports?state=open", nil, nil)
	var rid string
	if items, _ := reps["data"].([]any); code == 200 {
		for _, it := range items {
			if r := it.(map[string]any); r["target_id"] == pid {
				rid = fmt.Sprint(r["id"])
			}
		}
	}
	check("report appears in the admin queue", rid != "", raw)
	if rid == "" {
		return
	}
	code, res, raw := call("POST", admin+"/v1/admin/reports/"+rid+"/resolve", map[string]string{"resolution": "remove_content", "note": "smoke"},
		map[string]string{"X-VIRA-CSRF": "1"})
	check("resolve report (remove content)", code == 200 && res["state"] == "actioned", raw)
}
