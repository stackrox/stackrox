import requests
import tempfile
import os
import zipfile
import shutil

# Define the software versions as a list
software_versions = [
    "4.0.0",
    "4.0.x-295-gad49b77433"
]

# Base URL for downloading zip files
base_url = "https://github.com/stackrox/stackrox/archive/refs/tags/"

# Create a temporary directory to store the downloaded files
temp_dir = tempfile.mkdtemp()

# Iterate over the software versions and download the zip files
for version in software_versions:
    # Construct the download URL for the current version
    download_url = f"{base_url}{version}.zip"

    # Specify the file path to save the downloaded zip file in the version directory
    file_path = os.path.join(temp_dir, f"{version}.zip")

    # Send a GET request to download the zip file
    response = requests.get(download_url)

    # Check if the request was successful (status code 200)
    if response.status_code == 200:
        try:
            # Save the zip file to the temporary directory
            with open(file_path, "wb") as file:
                file.write(response.content)
            print(f"Successfully downloaded {file_path}")

            # Extract the contents of the zip file
            with zipfile.ZipFile(file_path, 'r') as zip_ref:
                # Extract all files to the version directory
                zip_ref.extractall(temp_dir)
            print(f"Successfully extracted files from {file_path}")

            # Change directory to the stackrox_dir folder
            stackrox_dir = os.path.join(temp_dir, f"stackrox-{version}")
            os.chdir(stackrox_dir)

            os.chdir("scanner")
            # List all files in the stackrox_dir folder
            file_list = os.listdir(os.path.join(stackrox_dir, "scanner"))
            print("Files in stackrox/scanner folder:", file_list)

            os.system("go run cmd/updater/update.go")
            os.system("mv tmp " + version)
            os.system("gsutil cp -r " + version + " gs://scanner-v4-test/")

            # Remove the stackrox_dir and its contents
            if os.path.isdir(stackrox_dir):
                shutil.rmtree(stackrox_dir)
        except Exception as e:
            print(f"An error occurred: {str(e)}")
            continue
    else:
        print(f"Failed to download {file_path}. Status code: {response.status_code}")

print("Temporary directory:", temp_dir)
