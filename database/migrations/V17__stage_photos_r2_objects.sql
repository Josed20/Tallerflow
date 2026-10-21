ALTER TABLE stage_photos ADD COLUMN object_key text;
ALTER TABLE stage_photos ALTER COLUMN photo_data DROP NOT NULL;
ALTER TABLE stage_photos ADD CONSTRAINT stage_photos_storage_check
    CHECK ((photo_data IS NOT NULL AND object_key IS NULL) OR
           (photo_data IS NULL AND object_key IS NOT NULL));
CREATE UNIQUE INDEX stage_photos_object_key_idx ON stage_photos(object_key) WHERE object_key IS NOT NULL;
