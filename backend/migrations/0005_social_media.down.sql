DROP TABLE social.comments, social.likes, social.post_media, social.posts, social.blocks, social.follow_requests,
  social.follows, social.profiles, media.assets;
REVOKE USAGE ON SCHEMA social, media FROM app_public, app_admin, app_worker;
