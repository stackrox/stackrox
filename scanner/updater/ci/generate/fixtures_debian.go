package main

import (
	claircore "github.com/quay/claircore"
)

// debian.json.zst native fixture relationships, transcribed from the checked-in CI bundle.
// Kept independent from expected results in scanner/e2etests and backend QA.
// Scenarios: TestImage qa:debian-package-removal, distroless/static-debian11,
// and debian:12.0. Retain source-package identity for Debian matching.
var debianFixtures = []operation{
	{Member: "debian.json.zst", Updater: "debian/updater", Vulnerabilities: []*claircore.Vulnerability{
		{Updater: "debian/updater", Name: "CVE-2019-2201", Description: debianValue0, Links: "https://security-tracker.debian.org/tracker/CVE-2019-2201", Severity: "low", NormalizedSeverity: 3, Package: &debianValue1, Dist: &debianValue2, FixedInVersion: "1:2.0.5-1"},
		{Updater: "debian/updater", Name: "CVE-2019-2201", Description: debianValue0, Links: "https://security-tracker.debian.org/tracker/CVE-2019-2201", Severity: "low", NormalizedSeverity: 3, Package: &debianValue1, Dist: &debianValue3, FixedInVersion: "1:2.0.5-1"},
		{Updater: "debian/updater", Name: "CVE-2019-5436", Description: debianValue4, Links: "https://security-tracker.debian.org/tracker/CVE-2019-5436", Severity: "not yet assigned", Package: &debianValue5, Dist: &debianValue3, FixedInVersion: "7.64.0-4"},
		{Updater: "debian/updater", Name: "CVE-2019-5436", Description: debianValue4, Links: "https://security-tracker.debian.org/tracker/CVE-2019-5436", Severity: "not yet assigned", Package: &debianValue5, Dist: &debianValue2, FixedInVersion: "7.64.0-4"},
		{Updater: "debian/updater", Name: "CVE-2023-38546", Description: debianValue6, Links: "https://security-tracker.debian.org/tracker/CVE-2023-38546", Severity: "not yet assigned", Package: &debianValue5, Dist: &debianValue3, FixedInVersion: "8.3.0-3"},
		{Updater: "debian/updater", Name: "CVE-2023-38546", Description: debianValue6, Links: "https://security-tracker.debian.org/tracker/CVE-2023-38546", Severity: "not yet assigned", Package: &debianValue5, Dist: &debianValue2, FixedInVersion: "7.88.1-10+deb12u4"},
		{Updater: "debian/updater", Name: "CVE-2023-38545", Description: debianValue7, Links: "https://security-tracker.debian.org/tracker/CVE-2023-38545", Severity: "not yet assigned", Package: &debianValue5, Dist: &debianValue3, FixedInVersion: "8.3.0-3"},
		{Updater: "debian/updater", Name: "CVE-2023-38545", Description: debianValue7, Links: "https://security-tracker.debian.org/tracker/CVE-2023-38545", Severity: "not yet assigned", Package: &debianValue5, Dist: &debianValue2, FixedInVersion: "7.88.1-10+deb12u4"},
		{Updater: "debian/updater", Name: "CVE-2021-33910", Description: debianValue8, Links: "https://security-tracker.debian.org/tracker/CVE-2021-33910", Severity: "not yet assigned", Package: &debianValue9, Dist: &debianValue2, FixedInVersion: "247.3-6"},
		{Updater: "debian/updater", Name: "CVE-2021-33910", Description: debianValue8, Links: "https://security-tracker.debian.org/tracker/CVE-2021-33910", Severity: "not yet assigned", Package: &debianValue9, Dist: &debianValue3, FixedInVersion: "247.3-6"},
		{Updater: "debian/updater", Name: "CVE-2023-7008", Description: debianValue10, Links: "https://security-tracker.debian.org/tracker/CVE-2023-7008", Severity: "not yet assigned", Package: &debianValue9, Dist: &debianValue2, FixedInVersion: "252.21-1~deb12u1"},
		{Updater: "debian/updater", Name: "CVE-2023-7008", Description: debianValue10, Links: "https://security-tracker.debian.org/tracker/CVE-2023-7008", Severity: "not yet assigned", Package: &debianValue9, Dist: &debianValue3, FixedInVersion: "255.1-3"},
		{Updater: "debian/updater", Name: "CVE-2019-9704", Description: debianValue11, Links: "https://security-tracker.debian.org/tracker/CVE-2019-9704", Severity: "low", NormalizedSeverity: 3, Package: &debianValue12, Dist: &debianValue2, FixedInVersion: "3.0pl1-133"},
		{Updater: "debian/updater", Name: "CVE-2019-9704", Description: debianValue11, Links: "https://security-tracker.debian.org/tracker/CVE-2019-9704", Severity: "low", NormalizedSeverity: 3, Package: &debianValue12, Dist: &debianValue3, FixedInVersion: "3.0pl1-133"},
		{Updater: "debian/updater", Name: "CVE-2019-9706", Description: debianValue13, Links: "https://security-tracker.debian.org/tracker/CVE-2019-9706", Severity: "not yet assigned", Package: &debianValue12, Dist: &debianValue3, FixedInVersion: "3.0pl1-133"},
		{Updater: "debian/updater", Name: "CVE-2019-9706", Description: debianValue13, Links: "https://security-tracker.debian.org/tracker/CVE-2019-9706", Severity: "not yet assigned", Package: &debianValue12, Dist: &debianValue2, FixedInVersion: "3.0pl1-133"},
		{Updater: "debian/updater", Name: "CVE-2017-9525", Description: debianValue14, Links: "https://security-tracker.debian.org/tracker/CVE-2017-9525", Severity: "not yet assigned", Package: &debianValue12, Dist: &debianValue3, FixedInVersion: "3.0pl1-129"},
		{Updater: "debian/updater", Name: "CVE-2017-9525", Description: debianValue14, Links: "https://security-tracker.debian.org/tracker/CVE-2017-9525", Severity: "not yet assigned", Package: &debianValue12, Dist: &debianValue2, FixedInVersion: "3.0pl1-129"},
		{Updater: "debian/updater", Name: "CVE-2019-9705", Description: debianValue15, Links: "https://security-tracker.debian.org/tracker/CVE-2019-9705", Severity: "low", NormalizedSeverity: 3, Package: &debianValue12, Dist: &debianValue2, FixedInVersion: "3.0pl1-133"},
		{Updater: "debian/updater", Name: "CVE-2019-9705", Description: debianValue15, Links: "https://security-tracker.debian.org/tracker/CVE-2019-9705", Severity: "low", NormalizedSeverity: 3, Package: &debianValue12, Dist: &debianValue3, FixedInVersion: "3.0pl1-133"},
		{Updater: "debian/updater", Name: "CVE-2023-3138", Description: debianValue16, Links: "https://security-tracker.debian.org/tracker/CVE-2023-3138", Severity: "not yet assigned", Package: &debianValue17, Dist: &debianValue2, FixedInVersion: "2:1.8.4-2+deb12u1"},
		{Updater: "debian/updater", Name: "CVE-2023-3138", Description: debianValue16, Links: "https://security-tracker.debian.org/tracker/CVE-2023-3138", Severity: "not yet assigned", Package: &debianValue17, Dist: &debianValue3, FixedInVersion: "2:1.8.6-1"},
		{Updater: "debian/updater", Name: "CVE-2021-3520", Description: debianValue18, Links: "https://security-tracker.debian.org/tracker/CVE-2021-3520", Severity: "not yet assigned", Package: &debianValue19, Dist: &debianValue2, FixedInVersion: "1.9.3-2"},
		{Updater: "debian/updater", Name: "CVE-2021-3520", Description: debianValue18, Links: "https://security-tracker.debian.org/tracker/CVE-2021-3520", Severity: "not yet assigned", Package: &debianValue19, Dist: &debianValue3, FixedInVersion: "1.9.3-2"},
		{Updater: "debian/updater", Name: "CVE-2021-28831", Description: debianValue20, Links: "https://security-tracker.debian.org/tracker/CVE-2021-28831", Severity: "not yet assigned", Package: &debianValue21, Dist: &debianValue2, FixedInVersion: "1:1.35.0-1"},
		{Updater: "debian/updater", Name: "CVE-2021-28831", Description: debianValue20, Links: "https://security-tracker.debian.org/tracker/CVE-2021-28831", Severity: "not yet assigned", Package: &debianValue21, Dist: &debianValue3, FixedInVersion: "1:1.35.0-1"},
		{Updater: "debian/updater", Name: "CVE-2022-30065", Description: debianValue22, Links: "https://security-tracker.debian.org/tracker/CVE-2022-30065", Severity: "unimportant", NormalizedSeverity: 2, Package: &debianValue21, Dist: &debianValue3, FixedInVersion: "1:1.36.1-1"},
		{Updater: "debian/updater", Name: "CVE-2022-30065", Description: debianValue22, Links: "https://security-tracker.debian.org/tracker/CVE-2022-30065", Severity: "unimportant", NormalizedSeverity: 2, Package: &debianValue21, Dist: &debianValue2},
		{Updater: "debian/updater", Name: "CVE-2022-3602", Description: debianValue23, Links: "https://security-tracker.debian.org/tracker/CVE-2022-3602", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue3, FixedInVersion: "3.0.7-1"},
		{Updater: "debian/updater", Name: "CVE-2022-3602", Description: debianValue23, Links: "https://security-tracker.debian.org/tracker/CVE-2022-3602", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue2, FixedInVersion: "3.0.7-1"},
		{Updater: "debian/updater", Name: "CVE-2022-3786", Description: debianValue25, Links: "https://security-tracker.debian.org/tracker/CVE-2022-3786", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue2, FixedInVersion: "3.0.7-1"},
		{Updater: "debian/updater", Name: "CVE-2022-3786", Description: debianValue25, Links: "https://security-tracker.debian.org/tracker/CVE-2022-3786", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue3, FixedInVersion: "3.0.7-1"},
		{Updater: "debian/updater", Name: "CVE-2025-15467", Description: debianValue26, Links: "https://security-tracker.debian.org/tracker/CVE-2025-15467", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue2, FixedInVersion: "3.0.18-1~deb12u2"},
		{Updater: "debian/updater", Name: "CVE-2025-15467", Description: debianValue26, Links: "https://security-tracker.debian.org/tracker/CVE-2025-15467", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue3, FixedInVersion: "3.5.4-1~deb13u2"},
		{Updater: "debian/updater", Name: "CVE-2018-0735", Description: debianValue27, Links: "https://security-tracker.debian.org/tracker/CVE-2018-0735", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue2, FixedInVersion: "1.1.1a-1"},
		{Updater: "debian/updater", Name: "CVE-2018-0735", Description: debianValue27, Links: "https://security-tracker.debian.org/tracker/CVE-2018-0735", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue3, FixedInVersion: "1.1.1a-1"},
		{Updater: "debian/updater", Name: "CVE-2022-2097", Description: debianValue28, Links: "https://security-tracker.debian.org/tracker/CVE-2022-2097", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue3, FixedInVersion: "3.0.5-1"},
		{Updater: "debian/updater", Name: "CVE-2022-2097", Description: debianValue28, Links: "https://security-tracker.debian.org/tracker/CVE-2022-2097", Severity: "not yet assigned", Package: &debianValue24, Dist: &debianValue2, FixedInVersion: "3.0.5-1"},
		{Updater: "debian/updater", Name: "CVE-2019-11068", Description: debianValue29, Links: "https://security-tracker.debian.org/tracker/CVE-2019-11068", Severity: "not yet assigned", Package: &debianValue30, Dist: &debianValue2, FixedInVersion: "1.1.32-2.1"},
		{Updater: "debian/updater", Name: "CVE-2019-11068", Description: debianValue29, Links: "https://security-tracker.debian.org/tracker/CVE-2019-11068", Severity: "not yet assigned", Package: &debianValue30, Dist: &debianValue3, FixedInVersion: "1.1.32-2.1"},
		{Updater: "debian/updater", Name: "CVE-2023-4911", Description: debianValue31, Links: "https://security-tracker.debian.org/tracker/CVE-2023-4911", Severity: "not yet assigned", Package: &debianValue32, Dist: &debianValue3, FixedInVersion: "2.37-12"},
		{Updater: "debian/updater", Name: "CVE-2023-4911", Description: debianValue31, Links: "https://security-tracker.debian.org/tracker/CVE-2023-4911", Severity: "not yet assigned", Package: &debianValue32, Dist: &debianValue2, FixedInVersion: "2.36-9+deb12u3"},
		{Updater: "debian/updater", Name: "CVE-2011-3374", Description: debianValue33, Links: "https://security-tracker.debian.org/tracker/CVE-2011-3374", Severity: "unimportant", NormalizedSeverity: 2, Package: &debianValue34, Dist: &debianValue3},
		{Updater: "debian/updater", Name: "CVE-2011-3374", Description: debianValue33, Links: "https://security-tracker.debian.org/tracker/CVE-2011-3374", Severity: "unimportant", NormalizedSeverity: 2, Package: &debianValue34, Dist: &debianValue2},
		{Updater: "debian/updater", Name: "CVE-2017-1000382", Description: debianValue35, Links: "https://security-tracker.debian.org/tracker/CVE-2017-1000382", Severity: "unimportant", NormalizedSeverity: 2, Package: &debianValue36, Dist: &debianValue2},
		{Updater: "debian/updater", Name: "CVE-2017-1000382", Description: debianValue35, Links: "https://security-tracker.debian.org/tracker/CVE-2017-1000382", Severity: "unimportant", NormalizedSeverity: 2, Package: &debianValue36, Dist: &debianValue3},
	}},
}
var debianValue0 = "In generate_jsimd_ycc_rgb_convert_neon of jsimd_arm64_neon.S, there is a possible out of bounds write due to a missing bounds check. This could lead to remote code execution in an unprivileged process with no additional execution privileges needed. User interaction is needed for exploitation.Product: AndroidVersions: Android-8.0 Android-8.1 Android-9 Android-10Android ID: A-120551338"

var debianValue1 = claircore.Package{Name: "libjpeg-turbo", Kind: 1}

var debianValue2 = claircore.Distribution{DID: "debian", Name: "Debian GNU/Linux", Version: "12 (bookworm)", VersionCodeName: "bookworm", VersionID: "12", PrettyName: "Debian GNU/Linux 12 (bookworm)"}

var debianValue3 = claircore.Distribution{DID: "debian", Name: "Debian GNU/Linux", Version: "13 (trixie)", VersionCodeName: "trixie", VersionID: "13", PrettyName: "Debian GNU/Linux 13 (trixie)"}

var debianValue4 = "A heap buffer overflow in the TFTP receiving code allows for DoS or arbitrary code execution in libcurl versions 7.19.4 through 7.64.1."

var debianValue5 = claircore.Package{Name: "curl", Kind: 1}

var debianValue6 = "This flaw allows an attacker to insert cookies at will into a running program using libcurl, if the specific series of conditions are met.  libcurl performs transfers. In its API, an application creates \"easy handles\" that are the individual handles for single transfers.  libcurl provides a function call that duplicates en easy handle called [curl_easy_duphandle](https://curl.se/libcurl/c/curl_easy_duphandle.html).  If a transfer has cookies enabled when the handle is duplicated, the cookie-enable state is also cloned - but without cloning the actual cookies. If the source handle did not read any cookies from a specific file on disk, the cloned version of the handle would instead store the file name as `none` (using the four ASCII letters, no quotes).  Subsequent use of the cloned handle that does not explicitly set a source to load cookies from would then inadvertently load cookies from a file named `none` - if such a file exists and is readable in the current directory of the program using libcurl. And if using the correct file format of course."

var debianValue7 = "This flaw makes curl overflow a heap based buffer in the SOCKS5 proxy handshake.  When curl is asked to pass along the host name to the SOCKS5 proxy to allow that to resolve the address instead of it getting done by curl itself, the maximum length that host name can be is 255 bytes.  If the host name is detected to be longer, curl switches to local name resolving and instead passes on the resolved address only. Due to this bug, the local variable that means \"let the host resolve the name\" could get the wrong value during a slow SOCKS5 handshake, and contrary to the intention, copy the too long host name to the target buffer instead of copying just the resolved address there.  The target buffer being a heap based buffer, and the host name coming from the URL that curl has been told to operate with."

var debianValue8 = "basic/unit-name.c in systemd prior to 246.15, 247.8, 248.5, and 249.1 has a Memory Allocation with an Excessive Size Value (involving strdupa and alloca for a pathname controlled by a local attacker) that results in an operating system crash."

var debianValue9 = claircore.Package{Name: "systemd", Kind: 1}

var debianValue10 = "A vulnerability was found in systemd-resolved. This issue may allow systemd-resolved to accept records of DNSSEC-signed domains even when they have no signature, allowing man-in-the-middles (or the upstream DNS resolver) to manipulate records."

var debianValue11 = "Vixie Cron before the 3.0pl1-133 Debian package allows local users to cause a denial of service (daemon crash) via a large crontab file because the calloc return value is not checked."

var debianValue12 = claircore.Package{Name: "cron", Kind: 1}

var debianValue13 = "Vixie Cron before the 3.0pl1-133 Debian package allows local users to cause a denial of service (use-after-free and daemon crash) because of a force_rescan_user error."

var debianValue14 = "In the cron package through 3.0pl1-128 on Debian, and through 3.0pl1-128ubuntu2 on Ubuntu, the postinst maintainer script allows for group-crontab-to-root privilege escalation via symlink attacks against unsafe usage of the chown and chmod programs."

var debianValue15 = "Vixie Cron before the 3.0pl1-133 Debian package allows local users to cause a denial of service (memory consumption) via a large crontab file because an unlimited number of lines is accepted."

var debianValue16 = "A vulnerability was found in libX11. The security flaw occurs because the functions in src/InitExt.c in libX11 do not check that the values provided for the Request, Event, or Error IDs are within the bounds of the arrays that those functions write to, using those IDs as array indexes. They trust that they were called with values provided by an Xserver adhering to the bounds specified in the X11 protocol, as all X servers provided by X.Org do. As the protocol only specifies a single byte for these values, an out-of-bounds value provided by a malicious server (or a malicious proxy-in-the-middle) can only overwrite other portions of the Display structure and not write outside the bounds of the Display structure itself, possibly causing the client to crash with this memory corruption."

var debianValue17 = claircore.Package{Name: "libx11", Kind: 1}

var debianValue18 = "There's a flaw in lz4. An attacker who submits a crafted file to an application linked with lz4 may be able to trigger an integer overflow, leading to calling of memmove() on a negative size argument, causing an out-of-bounds write and/or a crash. The greatest impact of this flaw is to availability, with some potential impact to confidentiality and integrity as well."

var debianValue19 = claircore.Package{Name: "lz4", Kind: 1}

var debianValue20 = "decompress_gunzip.c in BusyBox through 1.32.1 mishandles the error bit on the huft_build result pointer, with a resultant invalid free or segmentation fault, via malformed gzip data."

var debianValue21 = claircore.Package{Name: "busybox", Kind: 1}

var debianValue22 = "A use-after-free in Busybox 1.35-x's awk applet leads to denial of service and possibly code execution when processing a crafted awk pattern in the copyvar function."

var debianValue23 = "A buffer overrun can be triggered in X.509 certificate verification, specifically in name constraint checking. Note that this occurs after certificate chain signature verification and requires either a CA to have signed the malicious certificate or for the application to continue certificate verification despite failure to construct a path to a trusted issuer. An attacker can craft a malicious email address to overflow four attacker-controlled bytes on the stack. This buffer overflow could result in a crash (causing a denial of service) or potentially remote code execution. Many platforms implement stack overflow protections which would mitigate against the risk of remote code execution. The risk may be further mitigated based on stack layout for any given platform/compiler. Pre-announcements of CVE-2022-3602 described this issue as CRITICAL. Further analysis based on some of the mitigating factors described above have led this to be downgraded to HIGH. Users are still encouraged to upgrade to a new version as soon as possible. In a TLS client, this can be triggered by connecting to a malicious server. In a TLS server, this can be triggered if the server requests client authentication and a malicious client connects. Fixed in OpenSSL 3.0.7 (Affected 3.0.0,3.0.1,3.0.2,3.0.3,3.0.4,3.0.5,3.0.6)."

var debianValue24 = claircore.Package{Name: "openssl", Kind: 1}

var debianValue25 = "A buffer overrun can be triggered in X.509 certificate verification, specifically in name constraint checking. Note that this occurs after certificate chain signature verification and requires either a CA to have signed a malicious certificate or for an application to continue certificate verification despite failure to construct a path to a trusted issuer. An attacker can craft a malicious email address in a certificate to overflow an arbitrary number of bytes containing the `.' character (decimal 46) on the stack. This buffer overflow could result in a crash (causing a denial of service). In a TLS client, this can be triggered by connecting to a malicious server. In a TLS server, this can be triggered if the server requests client authentication and a malicious client connects."

var debianValue26 = "Issue summary: Parsing CMS AuthEnvelopedData or EnvelopedData message with maliciously crafted AEAD parameters can trigger a stack buffer overflow.  Impact summary: A stack buffer overflow may lead to a crash, causing Denial of Service, or potentially remote code execution.  When parsing CMS (Auth)EnvelopedData structures that use AEAD ciphers such as AES-GCM, the IV (Initialization Vector) encoded in the ASN.1 parameters is copied into a fixed-size stack buffer without verifying that its length fits the destination. An attacker can supply a crafted CMS message with an oversized IV, causing a stack-based out-of-bounds write before any authentication or tag verification occurs.  Applications and services that parse untrusted CMS or PKCS#7 content using AEAD ciphers (e.g., S/MIME (Auth)EnvelopedData with AES-GCM) are vulnerable. Because the overflow occurs prior to authentication, no valid key material is required to trigger it. While exploitability to remote code execution depends on platform and toolchain mitigations, the stack-based write primitive represents a severe risk.  The FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this issue, as the CMS implementation is outside the OpenSSL FIPS module boundary.  OpenSSL 3.6, 3.5, 3.4, 3.3 and 3.0 are vulnerable to this issue.  OpenSSL 1.1.1 and 1.0.2 are not affected by this issue."

var debianValue27 = "The OpenSSL ECDSA signature algorithm has been shown to be vulnerable to a timing side channel attack. An attacker could use variations in the signing algorithm to recover the private key. Fixed in OpenSSL 1.1.0j (Affected 1.1.0-1.1.0i). Fixed in OpenSSL 1.1.1a (Affected 1.1.1)."

var debianValue28 = "AES OCB mode for 32-bit x86 platforms using the AES-NI assembly optimised implementation will not encrypt the entirety of the data under some circumstances. This could reveal sixteen bytes of data that was preexisting in the memory that wasn't written. In the special case of \"in place\" encryption, sixteen bytes of the plaintext would be revealed. Since OpenSSL does not support OCB based cipher suites for TLS and DTLS, they are both unaffected. Fixed in OpenSSL 3.0.5 (Affected 3.0.0-3.0.4). Fixed in OpenSSL 1.1.1q (Affected 1.1.1-1.1.1p)."

var debianValue29 = "libxslt through 1.1.33 allows bypass of a protection mechanism because callers of xsltCheckRead and xsltCheckWrite permit access even upon receiving a -1 error code. xsltCheckRead can return -1 for a crafted URL that is not actually invalid and is subsequently loaded."

var debianValue30 = claircore.Package{Name: "libxslt", Kind: 1}

var debianValue31 = "A buffer overflow was discovered in the GNU C Library's dynamic loader ld.so while processing the GLIBC_TUNABLES environment variable. This issue could allow a local attacker to use maliciously crafted GLIBC_TUNABLES environment variables when launching binaries with SUID permission to execute code with elevated privileges."

var debianValue32 = claircore.Package{Name: "glibc", Kind: 1}

var debianValue33 = "It was found that apt-key in apt, all versions, do not correctly validate gpg keys with the master keyring, leading to a potential man-in-the-middle attack."

var debianValue34 = claircore.Package{Name: "apt", Kind: 1}

var debianValue35 = "VIM version 8.0.1187 (and other versions most likely) ignores umask when creating a swap file (\"[ORIGINAL_FILENAME].swp\") resulting in files that may be world readable or otherwise accessible in ways not intended by the user running the vi binary."

var debianValue36 = claircore.Package{Name: "vim", Kind: 1}
