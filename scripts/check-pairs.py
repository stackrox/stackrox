#!/usr/bin/env python3
"""Check for mismatched _orig/_copy image pairs."""

import re
import sys
from collections import Counter

def check_pairs(input_file):
    orig_images = set()
    copy_images = set()
    orig_list = []  # Track all entries including duplicates
    copy_list = []
    
    with open(input_file, 'r') as f:
        for line in f:
            # Extract image name from the line
            match = re.search(r'"([^"]+)"', line)
            if match:
                image_name = match.group(1)
                if '_orig"' in line or image_name.endswith('_orig'):
                    # Extract base name without _orig
                    base = re.sub(r'_orig$', '', image_name)
                    orig_images.add(base)
                    orig_list.append(base)
                elif '_copy"' in line or image_name.endswith('_copy'):
                    # Extract base name without _copy
                    base = re.sub(r'_copy$', '', image_name)
                    copy_images.add(base)
                    copy_list.append(base)
    
    print(f"Total _orig entries: {len(orig_list)} (unique: {len(orig_images)})")
    print(f"Total _copy entries: {len(copy_list)} (unique: {len(copy_images)})")
    
    # Check for duplicates
    orig_counts = Counter(orig_list)
    copy_counts = Counter(copy_list)
    
    orig_dups = {k: v for k, v in orig_counts.items() if v > 1}
    copy_dups = {k: v for k, v in copy_counts.items() if v > 1}
    
    if orig_dups:
        print(f"\n=== Duplicate _orig images ({len(orig_dups)} names, {sum(orig_dups.values()) - len(orig_dups)} extra entries) ===")
        for img, count in sorted(orig_dups.items())[:20]:
            print(f"  {img} (appears {count} times)")
        if len(orig_dups) > 20:
            print(f"  ... and {len(orig_dups) - 20} more")
    
    if copy_dups:
        print(f"\n=== Duplicate _copy images ({len(copy_dups)} names, {sum(copy_dups.values()) - len(copy_dups)} extra entries) ===")
        for img, count in sorted(copy_dups.items())[:20]:
            print(f"  {img} (appears {count} times)")
        if len(copy_dups) > 20:
            print(f"  ... and {len(copy_dups) - 20} more")
    
    # Find mismatches
    orig_only = orig_images - copy_images
    copy_only = copy_images - orig_images
    
    if orig_only:
        print(f"\n=== Images with _orig but missing _copy ({len(orig_only)}) ===")
        for img in sorted(orig_only)[:20]:  # Show first 20
            print(f"  {img}")
        if len(orig_only) > 20:
            print(f"  ... and {len(orig_only) - 20} more")
    else:
        print("\n✓ All _orig images have matching _copy")
    
    if copy_only:
        print(f"\n=== Images with _copy but missing _orig ({len(copy_only)}) ===")
        for img in sorted(copy_only)[:20]:  # Show first 20
            print(f"  {img}")
        if len(copy_only) > 20:
            print(f"  ... and {len(copy_only) - 20} more")
    else:
        print("✓ All _copy images have matching _orig")
    
    if not orig_only and not copy_only:
        print("\n✅ All image pairs are complete!")
    
    return len(orig_only) + len(copy_only)

if __name__ == "__main__":
    input_file = sys.argv[1] if len(sys.argv) > 1 else "generated-images-entries.txt"
    sys.exit(check_pairs(input_file))
