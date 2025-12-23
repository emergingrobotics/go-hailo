#!/bin/bash
# Add Hailo repositories as git submodules for code inspection
# Usage: ./scripts/add-submodules.sh
#
# This script is idempotent - safe to run multiple times.
# It will add missing submodules and refresh existing ones.

set -e

REPOS_DIR="repos"

# Function to add or update a submodule (with branch tracking)
add_or_update_submodule() {
    local url="$1"
    local path="$2"
    local branch="$3"

    if [ -d "$path/.git" ] || [ -f "$path/.git" ]; then
        echo "  Updating $path (already exists)..."
        # Abort any in-progress merge
        git -C "$path" merge --abort 2>/dev/null || true
        # Clean up any conflicts
        git -C "$path" checkout -- . 2>/dev/null || true
        git -C "$path" fetch --all --prune
        if [ -n "$branch" ]; then
            git -C "$path" checkout "$branch"
            git -C "$path" reset --hard "origin/$branch"
        fi
    elif git config --file .gitmodules --get "submodule.$path.url" > /dev/null 2>&1; then
        echo "  Initializing $path (registered but not cloned)..."
        git submodule update --init "$path"
        if [ -n "$branch" ]; then
            git -C "$path" checkout "$branch"
        fi
    else
        echo "  Adding $path..."
        if [ -n "$branch" ]; then
            git submodule add -b "$branch" "$url" "$path"
        else
            git submodule add "$url" "$path"
        fi
    fi
}

# Function to add or update a submodule pinned to a specific tag (no merge)
add_or_update_submodule_tag() {
    local url="$1"
    local path="$2"
    local tag="$3"

    if [ -d "$path/.git" ] || [ -f "$path/.git" ]; then
        echo "  Updating $path to tag $tag (already exists)..."
        # Abort any in-progress merge
        git -C "$path" merge --abort 2>/dev/null || true
        # Clean up any conflicts
        git -C "$path" checkout -- . 2>/dev/null || true
        git -C "$path" fetch --all --tags --prune
        git -C "$path" checkout "$tag"
    elif git config --file .gitmodules --get "submodule.$path.url" > /dev/null 2>&1; then
        echo "  Initializing $path at tag $tag (registered but not cloned)..."
        git submodule update --init "$path"
        git -C "$path" fetch --tags
        git -C "$path" checkout "$tag"
    else
        echo "  Adding $path at tag $tag..."
        git submodule add "$url" "$path"
        git -C "$path" fetch --tags
        git -C "$path" checkout "$tag"
    fi
}

echo "Creating $REPOS_DIR directory..."
mkdir -p "$REPOS_DIR"

echo ""
echo "=== Essential repositories ==="

# hailort - Core runtime (hailo8 branch)
add_or_update_submodule \
    "https://github.com/hailo-ai/hailort.git" \
    "$REPOS_DIR/hailort" \
    "hailo8"

# hailort-drivers - PCIe drivers (hailo8 branch)
add_or_update_submodule \
    "https://github.com/hailo-ai/hailort-drivers.git" \
    "$REPOS_DIR/hailort-drivers" \
    "hailo8"

# hailo-rpi5-examples - RPi5 examples (main branch)
add_or_update_submodule \
    "https://github.com/hailo-ai/hailo-rpi5-examples.git" \
    "$REPOS_DIR/hailo-rpi5-examples" \
    "main"

# hailo_model_zoo - Model zoo (pinned to v2.17 tag for Hailo-8)
add_or_update_submodule_tag \
    "https://github.com/hailo-ai/hailo_model_zoo.git" \
    "$REPOS_DIR/hailo_model_zoo" \
    "v2.17"

echo ""
echo "=== Highly recommended repositories ==="

# tappas - GStreamer pipelines (master branch supports Hailo-8)
add_or_update_submodule \
    "https://github.com/hailo-ai/tappas.git" \
    "$REPOS_DIR/tappas" \
    "master"

# Hailo-Application-Code-Examples - C++/Python examples
add_or_update_submodule \
    "https://github.com/hailo-ai/Hailo-Application-Code-Examples.git" \
    "$REPOS_DIR/Hailo-Application-Code-Examples" \
    "main"

# hailo-apps-infra - Python infrastructure
add_or_update_submodule \
    "https://github.com/hailo-ai/hailo-apps-infra.git" \
    "$REPOS_DIR/hailo-apps-infra" \
    "main"

echo ""
echo "=== Optional/specialized repositories ==="

# hailo-CLIP - CLIP model support
add_or_update_submodule \
    "https://github.com/hailo-ai/hailo-CLIP.git" \
    "$REPOS_DIR/hailo-CLIP" \
    "main"

# hailo-BEV - Bird's Eye View perception
add_or_update_submodule \
    "https://github.com/hailo-ai/hailo-BEV.git" \
    "$REPOS_DIR/hailo-BEV" \
    "main"

# meta-hailo - Yocto layers (hailo8-kirkstone branch)
add_or_update_submodule \
    "https://github.com/hailo-ai/meta-hailo.git" \
    "$REPOS_DIR/meta-hailo" \
    "hailo8-kirkstone"

echo ""
echo "=== Finalizing ==="

echo "Initializing any nested submodules..."
git submodule update --init --recursive

echo ""
echo "Done! All submodules are up to date."
echo ""
echo "Submodules are located in: $REPOS_DIR/"
