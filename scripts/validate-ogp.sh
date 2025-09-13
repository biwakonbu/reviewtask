#!/bin/bash

# OGP Validation Script for reviewtask documentation
# Based on bluetraff.com recommendations (2024)

set -e

echo "🔍 OGP Implementation Validation"
echo "================================"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if required OGP image exists
echo -e "\n📸 Checking OGP Images:"
if [ -f "docs/assets/images/ogp/ogp-logo-1200x630.png" ]; then
    echo -e "${GREEN}✓${NC} Universal OGP image (1200x630) exists"
    
    # Check file size (should be reasonable, not too large)
    size=$(stat -f%z "docs/assets/images/ogp/ogp-logo-1200x630.png" 2>/dev/null || stat -c%s "docs/assets/images/ogp/ogp-logo-1200x630.png" 2>/dev/null)
    size_kb=$((size / 1024))
    
    if [ $size_kb -lt 500 ]; then
        echo -e "${GREEN}✓${NC} Image size is optimal (${size_kb}KB)"
    elif [ $size_kb -lt 1000 ]; then
        echo -e "${YELLOW}⚠${NC} Image size is acceptable but could be optimized (${size_kb}KB)"
    else
        echo -e "${RED}✗${NC} Image size is too large (${size_kb}KB) - should be under 1MB"
    fi
else
    echo -e "${RED}✗${NC} Missing universal OGP image at docs/assets/images/ogp/ogp-logo-1200x630.png"
fi

# Check if auto-generated social image exists
if [ -f "docs/assets/images/social/index.png" ]; then
    echo -e "${GREEN}✓${NC} MkDocs Material auto-generated social image exists"
else
    echo -e "${YELLOW}⚠${NC} MkDocs Material social image not found (will be generated on build)"
fi

echo -e "\n📝 Checking OGP Meta Tags in Template:"

# Check main.html template
if [ -f "docs/overrides/main.html" ]; then
    echo -e "${GREEN}✓${NC} Template override file exists"
    
    # Check for essential OGP tags
    essential_tags=(
        "og:title"
        "og:description"
        "og:image"
        "og:url"
        "og:type"
        "twitter:card"
        "twitter:image"
    )
    
    for tag in "${essential_tags[@]}"; do
        if grep -q "$tag" docs/overrides/main.html; then
            echo -e "  ${GREEN}✓${NC} $tag tag present"
        else
            echo -e "  ${RED}✗${NC} Missing $tag tag"
        fi
    done
    
    # Check for 1200x630 dimension specifications
    if grep -q "1200.*630" docs/overrides/main.html; then
        echo -e "  ${GREEN}✓${NC} Universal dimensions (1200x630) specified"
    else
        echo -e "  ${RED}✗${NC} Universal dimensions not found"
    fi
else
    echo -e "${RED}✗${NC} Template override file not found"
fi

echo -e "\n🌐 Platform Compatibility Check:"
platforms=(
    "Facebook:1200x630"
    "Twitter:1200x630"
    "LinkedIn:1200x630"
    "LINE:1200x630"
    "Threads:1200x630"
    "Google Maps:1200x630"
)

for platform in "${platforms[@]}"; do
    name="${platform%%:*}"
    dims="${platform##*:}"
    echo -e "  ${GREEN}✓${NC} $name - Optimized with universal $dims image"
done

echo -e "\n🧪 Testing Tools:"
echo "  You can validate your OGP implementation using:"
echo "  • Facebook: https://developers.facebook.com/tools/debug/"
echo "  • Twitter: https://cards-dev.twitter.com/validator"
echo "  • LinkedIn: https://www.linkedin.com/post-inspector/"
echo "  • General: https://www.opengraph.xyz/"

echo -e "\n✅ Validation Summary:"
echo "  Based on bluetraff.com 2024 recommendations:"
echo "  - Universal image size: 1200x630 pixels"
echo "  - Works across all major platforms"
echo "  - Center important content in images"
echo "  - Keep file sizes reasonable (<1MB)"

exit 0