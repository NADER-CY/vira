#!/usr/bin/env python3
"""Generates contracts/openapi/*.json (OpenAPI 3.1) — the API contract source of truth.

Edit this file, run `python3 scripts/gen_openapi.py`, commit both. CI fails if the JSON is stale
(`--check`), and the Go contract test fails if a route exists in code but not in the contract (or vice versa).
"""
import json, os, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CODES = ["validation_failed", "unauthenticated", "forbidden", "not_found", "method_not_allowed", "conflict",
         "idempotency_conflict", "request_in_progress", "precondition_failed", "precondition_required", "rate_limited",
         "feature_disabled", "feature_locked", "account_restricted", "mfa_required", "mfa_invalid", "maintenance",
         "upgrade_required", "internal_error", "service_unavailable"]

def ref(name): return {"$ref": f"#/components/schemas/{name}"}
def problem(desc): return {"description": desc, "content": {"application/problem+json": {"schema": ref("Problem")}}}
def ok(schema, desc="OK", headers=None):
    r = {"description": desc, "content": {"application/json": {"schema": schema}}}
    if headers: r["headers"] = headers
    return r

COMMON = {
    "Problem": {
        "type": "object", "description": "RFC 9457 problem details (doc 07 §1). Extension members may appear (e.g. gate, min_supported_version).",
        "required": ["type", "title", "status", "code"],
        "properties": {
            "type": {"type": "string", "example": "urn:vira:error:not_found"},
            "title": {"type": "string", "description": "Persian, user-facing"},
            "status": {"type": "integer"},
            "code": {"type": "string", "enum": CODES},
            "detail": {"type": "string"},
            "trace_id": {"type": "string"},
            "errors": {"type": "array", "items": {"type": "object", "required": ["field", "code"],
                       "properties": {"field": {"type": "string"}, "code": {"type": "string"}, "detail": {"type": "string"}}}},
        },
        "additionalProperties": True,
    },
    "Live": {"type": "object", "required": ["status", "service"], "properties": {
        "status": {"type": "string", "enum": ["ok"]}, "service": {"type": "string"}, "version": {"type": "string"}}},
    "Readiness": {"type": "object", "required": ["status", "service", "checks"], "properties": {
        "status": {"type": "string", "enum": ["ok", "fail"]}, "service": {"type": "string"},
        "checks": {"type": "array", "items": {"type": "object", "required": ["name", "status", "duration_ms"], "properties": {
            "name": {"type": "string"}, "status": {"type": "string", "enum": ["ok", "fail"]},
            "error": {"type": "string"}, "duration_ms": {"type": "integer"}}}}}},
}

def health_paths():
    return {
        "/health/live": {"get": {"operationId": "healthLive", "tags": ["health"], "summary": "Liveness",
                                 "responses": {"200": ok(ref("Live"))}}},
        "/health/ready": {"get": {"operationId": "healthReady", "tags": ["health"], "summary": "Readiness (dependency checks)",
                                  "responses": {"200": ok(ref("Readiness")), "503": ok(ref("Readiness"), "Not ready")}}},
    }

def public():
    paths = health_paths()
    paths["/v1/config"] = {"get": {"operationId": "getConfig", "tags": ["platform"],
        "summary": "Client configuration: non-regulated flags, regulated capabilities (always off), public remote config",
        "responses": {"200": ok(ref("ClientConfig"))}}}
    paths["/v1/me"] = {"get": {"operationId": "getMe", "tags": ["auth"], "security": [{"bearer": []}],
        "summary": "The authenticated principal",
        "parameters": [{"$ref": "#/components/parameters/AppVersion"}],
        "responses": {"200": ok(ref("Me")), "401": problem("Unauthenticated"), "426": problem("App too old"),
                      "503": problem("Maintenance")}}}
    schemas = dict(COMMON)
    schemas["ClientConfig"] = {"type": "object", "required": ["api_version", "flags", "regulated_locked", "config"], "properties": {
        "api_version": {"type": "string"}, "server_version": {"type": "string"},
        "flags": {"type": "object", "additionalProperties": {"type": "boolean"}},
        "regulated_locked": {"type": "object", "description": "Regulated capability → approval gate (FG-1b, G1, G2, G3)",
                             "additionalProperties": {"type": "string"}},
        "config": {"type": "object", "additionalProperties": True}}}
    schemas["Me"] = {"type": "object", "required": ["subject", "kind"], "properties": {
        "subject": {"type": "string"}, "kind": {"type": "string"},
        "scopes": {"type": ["array", "null"], "items": {"type": "string"}}, "expires_at": {"type": "string", "format": "date-time"}}}
    return {
        "openapi": "3.1.0",
        "info": {"title": "VIRA Public API", "version": "1.0.0-phase1",
                 "description": "Mobile app and seller web API (doc 07). Phase 1: platform surface only. "
                                "Unsafe methods accept Idempotency-Key; money is integer Rial."},
        "servers": [{"url": "http://localhost:8080", "description": "local development"}],
        "paths": paths,
        "components": {
            "schemas": schemas,
            "securitySchemes": {"bearer": {"type": "http", "scheme": "bearer", "bearerFormat": "JWT (ES256)"}},
            "parameters": {"AppVersion": {"name": "X-App-Version", "in": "header", "required": False,
                                          "schema": {"type": "string", "example": "1.0.0"},
                                          "description": "Mobile app version; below mobile.min_supported_version → 426"}},
        },
    }

def admin():
    paths = health_paths()
    sec = [{"session": []}]
    csrf = {"$ref": "#/components/parameters/CSRF"}
    paths["/v1/auth/login"] = {"post": {"operationId": "adminLogin", "tags": ["auth"], "summary": "Password step; returns an MFA token",
        "parameters": [csrf], "requestBody": {"required": True, "content": {"application/json": {"schema": ref("LoginRequest")}}},
        "responses": {"200": ok(ref("LoginResponse")), "401": problem("Invalid credentials"), "403": problem("Account restricted"),
                      "422": problem("Validation failed")}}}
    paths["/v1/auth/mfa"] = {"post": {"operationId": "adminMfa", "tags": ["auth"], "summary": "TOTP step; sets the session cookie",
        "parameters": [csrf], "requestBody": {"required": True, "content": {"application/json": {"schema": ref("MFARequest")}}},
        "responses": {"200": ok(ref("Admin"), headers={"Set-Cookie": {"schema": {"type": "string"},
                      "description": "vira_admin_session; HttpOnly; SameSite=Strict; Secure outside development"}}),
                      "401": problem("Invalid or expired")}}}
    paths["/v1/auth/logout"] = {"post": {"operationId": "adminLogout", "tags": ["auth"], "security": sec, "parameters": [csrf],
        "responses": {"204": {"description": "Logged out"}, "401": problem("Unauthenticated")}}}
    paths["/v1/auth/me"] = {"get": {"operationId": "adminMe", "tags": ["auth"], "security": sec,
        "responses": {"200": ok(ref("Admin")), "401": problem("Unauthenticated")}}}
    paths["/v1/admin/system/status"] = {"get": {"operationId": "systemStatus", "tags": ["system"], "security": sec,
        "x-permission": "system.read", "responses": {"200": ok(ref("SystemStatus")), "401": problem("Unauthenticated"), "403": problem("Forbidden")}}}
    paths["/v1/admin/flags"] = {"get": {"operationId": "listFlags", "tags": ["flags"], "security": sec, "x-permission": "flags.read",
        "responses": {"200": ok({"type": "object", "required": ["data"], "properties": {"data": {"type": "array", "items": ref("Flag")}}}),
                      "401": problem("Unauthenticated"), "403": problem("Forbidden")}}}
    paths["/v1/admin/flags/{key}"] = {"put": {"operationId": "setFlag", "tags": ["flags"], "security": sec, "x-permission": "flags.write",
        "summary": "Change a non-regulated flag. Regulated flags answer 409 feature_locked with their gate.",
        "parameters": [{"name": "key", "in": "path", "required": True, "schema": {"type": "string"}},
                       {"name": "If-Match", "in": "header", "required": True, "schema": {"type": "string"}, "description": "Current flag version"},
                       csrf],
        "requestBody": {"required": True, "content": {"application/json": {"schema": ref("SetFlagRequest")}}},
        "responses": {"200": ok(ref("Flag"), headers={"ETag": {"schema": {"type": "string"}}}), "401": problem("Unauthenticated"),
                      "403": problem("Forbidden"), "404": problem("Unknown flag"), "409": problem("Regulated (feature_locked)"),
                      "412": problem("Version mismatch"), "422": problem("Validation failed"), "428": problem("If-Match required")}}}
    paths["/v1/admin/config"] = {"get": {"operationId": "listConfig", "tags": ["config"], "security": sec, "x-permission": "config.read",
        "responses": {"200": ok({"type": "object", "required": ["data"], "properties": {"data": {"type": "array", "items": ref("ConfigItem")}}}),
                      "401": problem("Unauthenticated"), "403": problem("Forbidden")}}}
    paths["/v1/admin/audit-events"] = {"get": {"operationId": "listAuditEvents", "tags": ["audit"], "security": sec, "x-permission": "audit.read",
        "parameters": [{"name": n, "in": "query", "required": False, "schema": s} for n, s in
                       [("action", {"type": "string"}), ("actor_id", {"type": "string"}), ("cursor", {"type": "string"}),
                        ("limit", {"type": "integer", "minimum": 1, "maximum": 100})]],
        "responses": {"200": ok({"type": "object", "required": ["data", "next_cursor"], "properties": {
                          "data": {"type": "array", "items": ref("AuditEvent")}, "next_cursor": {"type": ["string", "null"]}}}),
                      "401": problem("Unauthenticated"), "403": problem("Forbidden"), "422": problem("Bad cursor")}}}
    paths["/v1/admin/roles"] = {"get": {"operationId": "listRoles", "tags": ["admins"], "security": sec, "x-permission": "admins.read",
        "responses": {"200": ok({"type": "object", "required": ["data"], "properties": {"data": {"type": "array", "items": ref("Role")}}}),
                      "401": problem("Unauthenticated"), "403": problem("Forbidden")}}}
    s = dict(COMMON)
    s["LoginRequest"] = {"type": "object", "required": ["email", "password"], "additionalProperties": False,
                         "properties": {"email": {"type": "string", "format": "email"}, "password": {"type": "string"}}}
    s["LoginResponse"] = {"type": "object", "required": ["mfa_required", "mfa_token", "mfa_expires_in"], "properties": {
        "mfa_required": {"type": "boolean"}, "mfa_token": {"type": "string"}, "mfa_expires_in": {"type": "integer"}}}
    s["MFARequest"] = {"type": "object", "required": ["mfa_token", "code"], "additionalProperties": False, "properties": {
        "mfa_token": {"type": "string"}, "code": {"type": "string", "pattern": "^[0-9]{6}$"}}}
    s["Admin"] = {"type": "object", "required": ["id", "email", "display_name", "roles", "permissions"], "properties": {
        "id": {"type": "string", "format": "uuid"}, "email": {"type": "string"}, "display_name": {"type": "string"},
        "roles": {"type": "array", "items": {"type": "string"}}, "permissions": {"type": "array", "items": {"type": "string"}},
        "session_expires_at": {"type": "string", "format": "date-time"}}}
    s["Flag"] = {"type": "object", "required": ["key", "description", "enabled", "regulated", "audiences", "version"], "properties": {
        "key": {"type": "string"}, "description": {"type": "string"}, "enabled": {"type": "boolean"}, "regulated": {"type": "boolean"},
        "gate": {"type": "string"}, "audiences": {"type": "array", "items": {"type": "string"}}, "version": {"type": "integer"},
        "updated_at": {"type": "string", "format": "date-time"}, "updated_by": {"type": "string"}}}
    s["SetFlagRequest"] = {"type": "object", "required": ["enabled", "reason"], "additionalProperties": False, "properties": {
        "enabled": {"type": "boolean"}, "reason": {"type": "string", "minLength": 3}}}
    s["ConfigItem"] = {"type": "object", "required": ["key", "value", "public"], "properties": {
        "key": {"type": "string"}, "value": {}, "public": {"type": "boolean"}, "updated_at": {"type": "string", "format": "date-time"}}}
    s["AuditEvent"] = {"type": "object", "required": ["id", "created_at", "actor_type", "action", "outcome"], "properties": {
        "id": {"type": "string"}, "created_at": {"type": "string", "format": "date-time"},
        "actor_type": {"type": "string", "enum": ["admin", "user", "seller_staff", "system", "anonymous"]},
        "actor_id": {"type": "string"}, "action": {"type": "string"}, "target_type": {"type": "string"}, "target_id": {"type": "string"},
        "outcome": {"type": "string", "enum": ["success", "failure", "denied"]}, "ip": {"type": "string"}, "user_agent": {"type": "string"},
        "trace_id": {"type": "string"}, "metadata": {"type": "object", "additionalProperties": True}}}
    s["Role"] = {"type": "object", "required": ["key", "name_fa", "permissions"], "properties": {
        "key": {"type": "string"}, "name_fa": {"type": "string"}, "description": {"type": "string"},
        "permissions": {"type": "array", "items": {"type": "string"}}}}
    s["SystemStatus"] = {"type": "object", "required": ["service", "readiness", "regulated_capabilities"], "properties": {
        "service": {"type": "string"}, "version": {"type": "string"}, "environment": {"type": "string"},
        "time": {"type": "string", "format": "date-time"}, "readiness": ref("Readiness"),
        "regulated_capabilities": {"type": "array", "items": {"type": "object", "required": ["key", "gate", "state"], "properties": {
            "key": {"type": "string"}, "gate": {"type": "string"}, "state": {"type": "string", "enum": ["locked_off"]}}}}}}
    return {
        "openapi": "3.1.0",
        "info": {"title": "VIRA Super Admin API", "version": "1.0.0-phase1",
                 "description": "Separate binary, DB role and auth realm (ADR-016). Cookie session + mandatory TOTP. "
                                "State-changing requests require the X-VIRA-CSRF: 1 header."},
        "servers": [{"url": "http://localhost:8081", "description": "local development"}],
        "paths": paths,
        "components": {"schemas": s,
            "securitySchemes": {"session": {"type": "apiKey", "in": "cookie", "name": "vira_admin_session"}},
            "parameters": {"CSRF": {"name": "X-VIRA-CSRF", "in": "header", "required": True, "schema": {"type": "string", "enum": ["1"]}}}},
    }

def dump(doc): return json.dumps(doc, ensure_ascii=False, indent=2, sort_keys=False) + "\n"

if __name__ == "__main__":
    out = {"contracts/openapi/public.v1.json": dump(public()), "contracts/openapi/admin.v1.json": dump(admin())}
    stale = []
    for rel, text in out.items():
        p = os.path.join(ROOT, rel)
        if "--check" in sys.argv:
            if not os.path.exists(p) or open(p, encoding="utf-8").read() != text:
                stale.append(rel)
        else:
            os.makedirs(os.path.dirname(p), exist_ok=True)
            open(p, "w", encoding="utf-8").write(text)
    if stale:
        print("stale generated contracts (run python3 scripts/gen_openapi.py):", *stale, sep="\n  ")
        sys.exit(1)
