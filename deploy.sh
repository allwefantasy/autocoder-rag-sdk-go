
#!/bin/bash

# Define source and target directories
SOURCE_DIR="/Users/allwefantasy/projects/auto-coder/rag-sdks/go"
TARGET_DIR="/Users/allwefantasy/projects/auto-coder-sdks/rag-sdks/go"

# Copy all files from source to target
echo "Copying files from $SOURCE_DIR to $TARGET_DIR"
rsync -av --delete --exclude='.git' "$SOURCE_DIR/" "$TARGET_DIR/"

# Change to target directory and push to GitHub
echo "Pushing changes to GitHub"
cd "$TARGET_DIR" || exit 1
git add .
git commit -m "Update SDK from auto-coder"
git push origin master

echo "Deployment completed successfully"
