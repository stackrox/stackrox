#!/usr/bin/env python3
"""Remove duplicate image entries, keeping only the first occurrence of each image name."""

import re
import sys

def dedupe_file(input_file):
    seen = set()
    output_lines = []
    duplicates = 0
    
    with open(input_file, 'r') as f:
        for line in f:
            line = line.rstrip('\n')
            if not line:
                continue
            
            # Extract image name (first quoted string)
            match = re.search(r'"([^"]+)"', line)
            if match:
                image_name = match.group(1)
                if image_name not in seen:
                    seen.add(image_name)
                    output_lines.append(line)
                else:
                    duplicates += 1
                    print(f"Removing duplicate: {image_name}")
            else:
                output_lines.append(line)
    
    # Write back
    with open(input_file, 'w') as f:
        for line in output_lines:
            f.write(line + '\n')
    
    print(f"\nTotal lines before: {len(output_lines) + duplicates}")
    print(f"Duplicates removed: {duplicates}")
    print(f"Total lines after: {len(output_lines)}")

if __name__ == "__main__":
    input_file = sys.argv[1] if len(sys.argv) > 1 else "generated-images-entries.txt"
    dedupe_file(input_file)
