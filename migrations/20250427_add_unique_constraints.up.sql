ALTER TABLE pet_categories ADD CONSTRAINT unique_pet_category_name UNIQUE (name);
ALTER TABLE pet_tags ADD CONSTRAINT unique_pet_tag_name UNIQUE (name);