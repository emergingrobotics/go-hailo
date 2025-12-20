#!/bin/bash
#
# get-orig-repos.sh
# Clone all Hailo-AI repositories needed for Hailo-8 M.2 + Raspberry Pi 5
#
# Usage: ./scripts/get-orig-repos.sh [target_directory]
#        Default target: ./repos
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Target directory for cloning
TARGET_DIR="${1:-repos}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Hailo-8 M.2 + Raspberry Pi 5 Repo Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Target directory: ${GREEN}${TARGET_DIR}${NC}"
echo ""

# Create target directory
mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"

# Function to clone and checkout
clone_repo() {
    local repo_url="$1"
    local branch="$2"
    local tag="$3"
    local repo_name=$(basename "$repo_url" .git)

    echo -e "${YELLOW}----------------------------------------${NC}"
    echo -e "${BLUE}Repository:${NC} $repo_name"

    if [ -d "$repo_name" ]; then
        echo -e "${YELLOW}  Already exists, skipping clone${NC}"
    else
        if [ -n "$branch" ]; then
            echo -e "${GREEN}  Cloning branch:${NC} $branch"
            git clone -b "$branch" "$repo_url"
        else
            echo -e "${GREEN}  Cloning default branch${NC}"
            git clone "$repo_url"
        fi
    fi

    # Checkout specific tag if provided
    if [ -n "$tag" ]; then
        echo -e "${GREEN}  Checking out tag:${NC} $tag"
        cd "$repo_name"
        git fetch --tags
        git checkout "$tag"
        cd ..
    fi

    echo -e "${GREEN}  Done${NC}"
}

echo -e "${BLUE}=== ESSENTIAL: Core Runtime & Drivers ===${NC}"
echo ""

# HailoRT - MUST use hailo8 branch (master is Hailo-10/15 only)
clone_repo "https://github.com/hailo-ai/hailort.git" "hailo8" ""

# HailoRT Drivers - MUST use hailo8 branch
clone_repo "https://github.com/hailo-ai/hailort-drivers.git" "hailo8" ""

echo ""
echo -e "${BLUE}=== ESSENTIAL: Raspberry Pi 5 Examples ===${NC}"
echo ""

# RPi5 Examples - main branch works for Hailo-8
clone_repo "https://github.com/hailo-ai/hailo-rpi5-examples.git" "" ""

echo ""
echo -e "${BLUE}=== ESSENTIAL: Model Zoo ===${NC}"
echo ""

# Model Zoo - MUST use v2.x tags (master is Hailo-10/15 only)
# Note: No v2.x branch exists, must use tags
clone_repo "https://github.com/hailo-ai/hailo_model_zoo.git" "" "v2.17"

echo ""
echo -e "${BLUE}=== HIGH PRIORITY: Application Frameworks ===${NC}"
echo ""

# TAPPAS - master supports both Hailo-8 and Hailo-10H
clone_repo "https://github.com/hailo-ai/tappas.git" "" ""

# Application Code Examples - main branch supports all Hailo devices
clone_repo "https://github.com/hailo-ai/Hailo-Application-Code-Examples.git" "" ""

# Apps Infrastructure - main branch supports RPi
clone_repo "https://github.com/hailo-ai/hailo-apps-infra.git" "" ""

echo ""
echo -e "${BLUE}=== OPTIONAL: Specialized Applications ===${NC}"
echo ""

# hailo-CLIP - Zero-shot classification (Hailo-8 supported)
clone_repo "https://github.com/hailo-ai/hailo-CLIP.git" "" ""

# hailo-BEV - Bird's Eye View (Hailo-8 supported)
clone_repo "https://github.com/hailo-ai/hailo-BEV.git" "" ""

echo ""
echo -e "${BLUE}=== SPECIALIZED: Yocto Integration ===${NC}"
echo ""

# meta-hailo - Use hailo8-kirkstone for Yocto Kirkstone builds
clone_repo "https://github.com/hailo-ai/meta-hailo.git" "hailo8-kirkstone" ""

echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}All repositories cloned successfully!${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""
echo -e "${BLUE}Summary of branch/tag selections:${NC}"
echo ""
echo "  hailort              -> hailo8 branch (v4.x for Hailo-8)"
echo "  hailort-drivers      -> hailo8 branch (v4.x for Hailo-8)"
echo "  hailo-rpi5-examples  -> main (default, supports Hailo-8)"
echo "  hailo_model_zoo      -> v2.17 tag (v2.x for Hailo-8)"
echo "  tappas               -> master (default, supports Hailo-8)"
echo "  Hailo-Application-Code-Examples -> main (default)"
echo "  hailo-apps-infra     -> main (default)"
echo "  hailo-CLIP           -> main (default, Hailo-8 specific)"
echo "  hailo-BEV            -> main (default, Hailo-8 specific)"
echo "  meta-hailo           -> hailo8-kirkstone (for Yocto)"
echo ""
echo -e "${BLUE}Repos are located in:${NC} $(pwd)"
echo ""
echo -e "${YELLOW}Note:${NC} For TAPPAS v5.1.0, ensure HailoRT v4.23.0 is installed"
echo ""
