-- base_images
CREATE TABLE IF NOT EXISTS base_images (
    id              BIGSERIAL PRIMARY KEY,
    registry        TEXT NOT NULL,
    repository      TEXT NOT NULL,
    tag             TEXT,
    digest          TEXT,
    config_digest   TEXT,
    created_at      TIMESTAMP DEFAULT now(),
    active          BOOLEAN,
    CONSTRAINT base_images_uniq UNIQUE (registry, repository, tag, digest)
);


-- BaseImageLayer
-- A relation linking base iamge to a layer
CREATE TABLE IF NOT EXISTS base_image_layer (
    id              BIGSERIAL PRIMARY KEY,
    iid             BIGINT NOT NULL REFERENCES base_images(id) ON DELETE CASCADE,
    layer_hash      TEXT NOT NULL,
    level           INTEGER NOT NULL,
    UNIQUE (iid, level)
);
