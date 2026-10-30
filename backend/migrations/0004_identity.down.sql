DROP TABLE identity.consents, identity.recovery_requests, identity.status_history, identity.refresh_tokens, identity.sessions,
  identity.devices, identity.otp_challenges, identity.phone_identities, identity.users;
REVOKE USAGE ON SCHEMA identity FROM app_public, app_admin, app_worker;
