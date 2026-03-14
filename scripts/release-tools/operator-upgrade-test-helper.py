#!/usr/bin/env -S uv run --quiet --script
# /// script
# dependencies = [
#   "selenium==4.27.1",
#   "beautifulsoup4==4.12.3",
#   "semver==3.0.2",
# ]
# ///

"""
Fetch Red Hat Advanced Cluster Security support policy and find compatible versions.

Takes a semver argument and returns the minimum RHACS version between oldest supported
and <major>.<minor-2>, along with the newest compatible OpenShift version.
"""

import re
import sys
import time
import semver
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from bs4 import BeautifulSoup


def fetch_page_with_selenium(url, wait_for_selector=None, wait_for_shadow_content=None):
    """
    Fetch a web page using Selenium WebDriver.

    Args:
        url: URL to fetch
        wait_for_selector: CSS selector to wait for (optional)
        wait_for_shadow_content: String to check for in shadow DOM (optional)

    Returns:
        Tuple of (html_content, shadow_content)
    """
    chrome_options = Options()
    chrome_options.add_argument("--headless=new")
    chrome_options.add_argument("--no-sandbox")
    chrome_options.add_argument("--disable-dev-shm-usage")
    chrome_options.add_argument("--disable-gpu")
    chrome_options.add_argument("--log-level=3")

    driver = webdriver.Chrome(options=chrome_options)

    try:
        driver.get(url)
        WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.TAG_NAME, "body"))
        )

        shadow_content = None

        if wait_for_selector:
            try:
                WebDriverWait(driver, 15).until(
                    EC.presence_of_element_located(
                        (By.CSS_SELECTOR, wait_for_selector))
                )
            except:
                pass

        if wait_for_shadow_content:
            for attempt in range(20):
                time.sleep(1)
                try:
                    script = """
                    let table = document.querySelector('plcc-table');
                    if (table && table.shadowRoot) {
                        return table.shadowRoot.innerHTML;
                    }
                    return null;
                    """
                    shadow_content = driver.execute_script(script)
                    if shadow_content and wait_for_shadow_content in shadow_content:
                        break
                except:
                    pass
        else:
            time.sleep(3)

        html_content = driver.page_source
        return html_content, shadow_content

    finally:
        driver.quit()


def fetch_rhacs_support_policy():
    """
    Fetch the RHACS support policy page from Red Hat.

    Returns:
        Tuple of (html_content, shadow_content)
    """
    url = "https://access.redhat.com/support/policy/updates/rhacs"
    return fetch_page_with_selenium(
        url,
        wait_for_selector="plcc-table",
        wait_for_shadow_content="product-lifecycle-Full Support"
    )


def fetch_compatibility_matrix():
    """
    Fetch the RHACS/OpenShift compatibility matrix page.

    Returns:
        HTML content as string
    """
    url = "https://access.redhat.com/articles/7045053"
    html_content, _ = fetch_page_with_selenium(url)
    return html_content


def expand_version_range(version_text):
    """
    Expand version ranges into individual versions.

    Args:
        version_text: String like '4.12 - 4.16' or '4.12, 4.14, 4.16-4.19'

    Returns:
        Sorted list of version strings
    """
    versions = set()
    version_text = re.sub(r'\[\d+\]', '', version_text)
    parts = [p.strip() for p in version_text.split(',')]

    for part in parts:
        range_match = re.match(r'(\d+\.\d+)\s*-\s*(\d+\.\d+)', part)
        if range_match:
            start_ver = range_match.group(1)
            end_ver = range_match.group(2)
            start_parts = [int(x) for x in start_ver.split('.')]
            end_parts = [int(x) for x in end_ver.split('.')]

            if start_parts[0] == end_parts[0]:
                for minor in range(start_parts[1], end_parts[1] + 1):
                    versions.add(f"{start_parts[0]}.{minor}")
        else:
            single_match = re.search(r'(\d+\.\d+)', part)
            if single_match:
                versions.add(single_match.group(1))

    return sorted(versions, key=lambda v: [int(x) for x in v.split('.')])


def parse_compatibility_matrix(html_content):
    """
    Parse the RHACS/OpenShift compatibility matrix from HTML.

    Args:
        html_content: HTML content as string

    Returns:
        Dict mapping RHACS versions to lists of compatible OpenShift versions
    """
    soup = BeautifulSoup(html_content, 'html.parser')
    tables = soup.find_all('table')
    compatibility = {}

    for table in tables:
        headers = table.find_all('th')
        if not headers:
            continue

        header_text = ' '.join([h.get_text(strip=True)
                               for h in headers]).lower()

        if 'openshift version compatible' in header_text:
            rows = table.find_all('tr')

            for row in rows:
                cells = row.find_all('td')
                if len(cells) < 2:
                    continue

                cell_texts = [c.get_text(strip=True) for c in cells]
                rhacs_match = re.search(
                    r'(?:ACS|RHACS)\s+([34]\.\d{1,2})', cell_texts[0])

                if rhacs_match:
                    rhacs_version = rhacs_match.group(1)
                    if len(cell_texts) >= 2:
                        openshift_versions = expand_version_range(
                            cell_texts[1])
                        if openshift_versions:
                            compatibility[rhacs_version] = openshift_versions

    return compatibility


def parse_all_versions(shadow_content):
    """
    Parse shadow DOM content to find all RHACS versions across all lifecycle phases.

    Args:
        shadow_content: Shadow DOM HTML content

    Returns:
        List of all version strings sorted by version number
    """
    if not shadow_content:
        return []

    version_row_pattern = r'<th\s+scope="row"\s+data-label="Version">\s*<!--[^>]*-->\s*(\d+\.\d+(?:\.\d+)?)'
    all_versions = re.findall(version_row_pattern, shadow_content)

    return sorted(
        set(all_versions),
        key=lambda v: [int(x) for x in v.split('.')]
    )


def parse_supported_versions(shadow_content):
    """
    Parse shadow DOM content to find actively supported RHACS versions.

    Args:
        shadow_content: Shadow DOM HTML content

    Returns:
        Tuple of (all_supported, full_support, maintenance) version lists
    """
    if not shadow_content:
        return [], [], []

    full_support_match = re.search(
        r'product-lifecycle-Full Support.*?</table>',
        shadow_content,
        re.DOTALL
    )

    maintenance_support_match = re.search(
        r'product-lifecycle-Maintenance Support.*?</table>',
        shadow_content,
        re.DOTALL
    )

    full_support_versions = []
    maintenance_versions = []
    version_row_pattern = r'<th\s+scope="row"\s+data-label="Version">\s*<!--[^>]*-->\s*(\d+\.\d+(?:\.\d+)?)'

    if full_support_match:
        full_support_text = full_support_match.group(0)
        full_support_versions = sorted(
            set(re.findall(version_row_pattern, full_support_text)),
            key=lambda v: [int(x) for x in v.split('.')]
        )

    if maintenance_support_match:
        maintenance_text = maintenance_support_match.group(0)
        maintenance_versions = sorted(
            set(re.findall(version_row_pattern, maintenance_text)),
            key=lambda v: [int(x) for x in v.split('.')]
        )

    all_supported = sorted(
        set(full_support_versions + maintenance_versions),
        key=lambda v: [int(x) for x in v.split('.')]
    )

    return all_supported, full_support_versions, maintenance_versions


def start_version(version, all_supported_versions, all_versions):
    """
    Get the start version for the test.

    The version the operator upgrade test will start from is determined
    by decrementing the minor part of the version by 2 and finding the
    smallest between <M>.<m-2>.0 and the oldest supported version.

    This logic can potentially give negative values when releasing new
    major versions, in this case, we take the oldest minor version of
    the previous major and we decrement it by 1 if the current minor is
    0. After applying this logic, we still find the minimum value
    between this and oldest supported version.

    Args:
        version: Input semver.Version
        all_supported_versions: List of actively supported version strings
        all_versions: List of all version strings (including EOL)

    Returns:
        Crafted semver.Version
    """
    oldest_supported = min([
        semver.Version.parse(v, optional_minor_and_patch=True)
        for v in all_supported_versions
    ])
    all_versions = [
        semver.Version.parse(v, optional_minor_and_patch=True)
        for v in all_versions
    ]

    crafted_major = version.major
    crafted_minor = version.minor - 2
    if crafted_minor < 0:
        newest_prev = max([
            v for v in all_versions if v.major == version.major - 1
        ])

        crafted_major = newest_prev.major
        crafted_minor = newest_prev.minor + crafted_minor + 1

    crafted = semver.Version(major=crafted_major, minor=crafted_minor, patch=0)
    return crafted if crafted < oldest_supported else oldest_supported


def find_compatible_openshift(target_version, start_version, compatibility_matrix):
    """
    Find the newest OpenShift version compatible with both RHACS versions.

    Args:
        target_version: RHACS being tested.
        start_version: RHACS version the upgrade test starts on
        compatibility_matrix: Dict mapping RHACS versions to OpenShift versions

    Returns:
        Newest compatible OpenShift version string, or None if no match found
    """
    target_version = f"{target_version.major}.{target_version.minor}"
    start_version = f"{start_version.major}.{start_version.minor}"

    target_ocp_version = set(compatibility_matrix.get(target_version, []))
    start_ocp_version = set(compatibility_matrix.get(start_version, []))

    # For minor or major releases no OCP compatibility is available,
    # use the newest version from the start version.
    if len(target_ocp_version) == 0:
        return max(start_ocp_version)

    common_versions = target_ocp_version & start_ocp_version

    if not common_versions:
        return None

    sorted_versions = sorted(common_versions, key=lambda v: [
                             int(x) for x in v.split('.')])
    return sorted_versions[-1]


def main():
    """Main function."""
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <version>", file=sys.stderr)
        print(f"Example: {sys.argv[0]} 4.12.3-rc.1", file=sys.stderr)
        sys.exit(1)

    input_version = sys.argv[1]

    try:
        target_version = semver.Version.parse(input_version)
    except ValueError as e:
        print(f"Error parsing version: {e}", file=sys.stderr)
        print(
            "Expected format: <major>.<minor>.<patch>[-rc.<RC>]", file=sys.stderr)
        sys.exit(1)

    _, shadow_content = fetch_rhacs_support_policy()
    all_supported, _, _ = parse_supported_versions(shadow_content)
    all_versions = parse_all_versions(shadow_content)

    if not all_supported:
        print("Error: No actively supported versions found.", file=sys.stderr)
        sys.exit(1)

    try:
        result = start_version(target_version, all_supported, all_versions)
    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    result_str = f"{result.major}.{result.minor}"

    compatibility_html = fetch_compatibility_matrix()
    compatibility_matrix = parse_compatibility_matrix(compatibility_html)
    compatible_ocp = find_compatible_openshift(
        target_version, result, compatibility_matrix)

    print(f"Upgrade start version: {result_str}")
    print(f"OCP version: {compatible_ocp if compatible_ocp else 'unknown'}")


if __name__ == "__main__":
    main()
