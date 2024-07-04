import json
import os
import argparse
from datetime import datetime
import requests
import gzip
import shutil

def dirpath(s):
    if os.path.isdir(s):
        return s
    raise ValueError(f"{s} is not a valid directory path")

def download_and_save_json(url, year, save_dir):
    try:
        response = requests.get(url, stream=True)
        response.raise_for_status()

        gz_file_path = os.path.join(save_dir, f"nvdcve-1.1-{year}.json.gz")
        json_file_path = os.path.join(save_dir, f"nvdcve-1.1-{year}.json")

        # Save the gzipped file
        with open(gz_file_path, 'wb') as gz_file:
            for chunk in response.iter_content(chunk_size=8192):
                gz_file.write(chunk)

        # Decompress and save the JSON file
        with gzip.open(gz_file_path, 'rb') as f_in:
            with open(json_file_path, 'wb') as f_out:
                shutil.copyfileobj(f_in, f_out)

        # Optionally, remove the gzipped file after decompressing
        os.remove(gz_file_path)
        print(f"Saved {json_file_path}")

    except Exception as e:
        print(f"Error processing {url} for year {year}: {e}")
        if os.path.exists(gz_file_path):
            os.remove(gz_file_path)
        if os.path.exists(json_file_path):
            os.remove(json_file_path)

def transform_json(input_file, output_file):
    with open(input_file, 'r') as f:
        data = json.load(f)

    transformed_items = []

    for item in data['CVE_Items']:
        cvss_v2_data = item.get('impact', {}).get('baseMetricV2', {}).get('cvssV2', {})
        transformed_item = {
            "cve": {
                "id": item['cve']['CVE_data_meta']['ID'],
                "published": item['publishedDate'],
                "lastModified": item['lastModifiedDate'],
                "descriptions": [{"lang": desc['lang'], "value": desc['value']} for desc in item['cve']['description']['description_data']],
                "metrics": {
                    "cvssMetricV2": [
                        {
                            "source": "nvd@nist.gov",
                            "type": "Primary",
                            "cvssData": {
                                "version": "2.0",
                                "vectorString": cvss_v2_data.get('vectorString', ''),
                                "accessVector": cvss_v2_data.get('accessVector', ''),
                                "accessComplexity": cvss_v2_data.get('accessComplexity', ''),
                                "authentication": cvss_v2_data.get('authentication', ''),
                                "confidentialityImpact": cvss_v2_data.get('confidentialityImpact', ''),
                                "integrityImpact": cvss_v2_data.get('integrityImpact', ''),
                                "availabilityImpact": cvss_v2_data.get('availabilityImpact', ''),
                                "baseScore": cvss_v2_data.get('baseScore', 0)
                            },
                            "baseSeverity": item.get('impact', {}).get('baseMetricV2', {}).get('severity', ''),
                            "exploitabilityScore": item.get('impact', {}).get('baseMetricV2', {}).get('exploitabilityScore', 0),
                            "impactScore": item.get('impact', {}).get('baseMetricV2', {}).get('impactScore', 0),
                            "acInsufInfo": item.get('impact', {}).get('baseMetricV2', {}).get('acInsufInfo', False),
                            "obtainAllPrivilege": item.get('impact', {}).get('baseMetricV2', {}).get('obtainAllPrivilege', False),
                            "obtainUserPrivilege": item.get('impact', {}).get('baseMetricV2', {}).get('obtainUserPrivilege', False),
                            "obtainOtherPrivilege": item.get('impact', {}).get('baseMetricV2', {}).get('obtainOtherPrivilege', False),
                            "userInteractionRequired": item.get('impact', {}).get('baseMetricV2', {}).get('userInteractionRequired', False)
                        }
                    ]
                },
                "configurations": [
                    {
                        "nodes": [
                            {
                                "operator": "OR",
                                "negate": False,
                                "cpeMatch": [
                                    {
                                        "vulnerable": cpe['vulnerable'],
                                        "criteria": cpe['cpe23Uri'],
                                        "matchCriteriaId": "D1A5AC77-6B76-41A9-8EFF-B5CA089313D4"  # Example matchCriteriaId
                                    } for cpe in item['configurations']['nodes'][0]['cpe_match']
                                ]
                            }
                        ]
                    }
                ],
                "references": [
                    {
                        "url": ref['url'],
                        "source": "cve@mitre.org",
                        "tags": ["Exploit", "Third Party Advisory"]  # Example tags
                    } for ref in item['cve']['references']['reference_data']
                ]
            }
        }
        transformed_items.append(transformed_item)

    with open(output_file, 'w') as f_out:
        for transformed_item in transformed_items:
            json.dump(transformed_item, f_out)
            f_out.write("\n")
        print(f"Saved {output_file}")

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        'dirpath',
        help="Path to directory where the NVD data will be saved.",
        type=dirpath
    )
    args = parser.parse_args()
    save_dir = args.dirpath

    base_url = "https://nvd.nist.gov/feeds/json/cve/1.1/"
    first_year = 2002
    current_year = datetime.now().year

    for year in range(first_year, current_year + 1):
        url = f"{base_url}nvdcve-1.1-{year}.json.gz"
        download_and_save_json(url, year, save_dir)
        json_file_path = os.path.join(save_dir, f"nvdcve-1.1-{year}.json")
        transformed_file_path = os.path.join(save_dir, f"{year}.nvd.json")
        transform_json(json_file_path, transformed_file_path)

if __name__ == "__main__":
    main()
