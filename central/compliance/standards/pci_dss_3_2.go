package standards

import "github.com/stackrox/rox/pkg/utils"

var pciDss3_2 = Standard{
	ID:   "PCI_DSS_3_2",
	Name: "PCI DSS 3.2",
	Categories: []Category{
		{
			ID:          "1",
			Name:        "1",
			Description: "Install and maintain a firewall configuration to protect cardholder data",
			Controls: []Control{
				{
					ID:          "1",
					Name:        "1",
					Description: "Install and maintain a firewall configuration to protect cardholder data",
				},
				{
					ID:          "1_1",
					Name:        "1.1",
					Description: "Establish and implement firewall and router configuration standards that include the following:",
				},
				{
					ID:          "1_1_1",
					Name:        "1.1.1",
					Description: "A formal process for approving and testing all network connections and changes to the firewall and router configurations",
				},
				{
					ID:          "1_1_2",
					Name:        "1.1.2",
					Description: "Current network diagram that identifies all connections between the cardholder data environment and other networks, including any wireless networks",
				},
				{
					ID:          "1_1_3",
					Name:        "1.1.3",
					Description: "Current diagram that shows all cardholder data flows across systems and networks",
				},
				{
					ID:          "1_1_4",
					Name:        "1.1.4",
					Description: "Requirements for a firewall at each Internet connection and between any demilitarized zone (DMZ) and the internal network zone",
				},
				{
					ID:          "1_1_5",
					Name:        "1.1.5",
					Description: "Description of groups, roles, and responsibilities for management of network components",
				},
				{
					ID:          "1_1_6",
					Name:        "1.1.6",
					Description: "Documentation of business justification and approval for use of all services, protocols, and ports allowed, including documentation of security features implemented for those protocols considered to be insecure.",
				},
				{
					ID:          "1_1_7",
					Name:        "1.1.7",
					Description: "Requirement to review firewall and router rule sets at least every six months",
				},
				{
					ID:          "1_2",
					Name:        "1.2",
					Description: "Build firewall and router configurations that restrict connections between untrusted networks and any system components in the cardholder data environment. Note: An “untrusted network” is any network that is external to the networks belonging to the entity under review, and/or which is out of the entity's ability to control or manage.",
				},
				{
					ID:          "1_2_1",
					Name:        "1.2.1",
					Description: "Restrict inbound and outbound traffic to that which is necessary for the cardholder data environment, and specifically deny all other traffic.",
				},
				{
					ID:          "1_2_2",
					Name:        "1.2.2",
					Description: "Secure and synchronize router configuration files.",
				},
				{
					ID:          "1_2_3",
					Name:        "1.2.3",
					Description: "Install perimeter firewalls between all wireless networks and the cardholder data environment, and configure these firewalls to deny or, if traffic is necessary for business purposes, permit only authorized traffic between the wireless environment and the cardholder data environment.",
				},
				{
					ID:          "1_3",
					Name:        "1.3",
					Description: "Prohibit direct public access between the Internet and any system component in the cardholder data environment.",
				},
				{
					ID:          "1_3_1",
					Name:        "1.3.1",
					Description: "Implement a DMZ to limit inbound traffic to only system components that provide authorized publicly accessible services, protocols, and ports.",
				},
				{
					ID:          "1_3_2",
					Name:        "1.3.2",
					Description: "Limit inbound Internet traffic to IP addresses within the DMZ.",
				},
				{
					ID:          "1_3_3",
					Name:        "1.3.3",
					Description: "Implement anti-spoofing measures to detect and block forged source IP addresses from entering the network. (For example, block traffic originating from the Internet with an internal source address.)",
				},
				{
					ID:          "1_3_4",
					Name:        "1.3.4",
					Description: "Do not allow unauthorized outbound traffic from the cardholder data environment to the Internet.",
				},
				{
					ID:          "1_3_5",
					Name:        "1.3.5",
					Description: "Permit only “established” connections into the network.",
				},
				{
					ID:          "1_3_6",
					Name:        "1.3.6",
					Description: "Place system components that store cardholder data (such as a database) in an internal network zone, segregated from the DMZ and other untrusted networks.",
				},
				{
					ID:          "1_3_7",
					Name:        "1.3.7",
					Description: "Do not disclose private IP addresses and routing information to unauthorized parties. Note: Methods to obscure IP addressing may include, but are not limited to: • Network Address Translation (NAT) • Placing servers containing cardholder data behind proxy servers/firewalls, • Removal or filtering of route advertisements for private networks that employ registered addressing, • Internal use of RFC1918 address space instead of registered addresses.",
				},
				{
					ID:          "1_4",
					Name:        "1.4",
					Description: "Install personal firewall software or equivalent functionality on any portable computing devices (including company and/or employee-owned) that connect to the Internet when outside the network (for example, laptops used by employees), and which are also used to access the CDE. Firewall (or equivalent) configurations include: • Specific configuration settings are defined. • Personal firewall (or equivalent functionality) is actively running. • Personal firewall (or equivalent functionality) is not alterable by users of the portable computing devices.",
				},
				{
					ID:          "1_5",
					Name:        "1.5",
					Description: "Ensure that security policies and operational procedures for managing firewalls are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "2",
			Name:        "2",
			Description: "Do not use vendor-supplied defaults for system passwords and other security parameters",
			Controls: []Control{
				{
					ID:          "2",
					Name:        "2",
					Description: "Do not use vendor-supplied defaults for system passwords and other security parameters",
				},
				{
					ID:          "2_1",
					Name:        "2.1",
					Description: "Always change vendor-supplied defaults and remove or disable unnecessary default accounts before installing a system on the network. This applies to ALL default passwords, including but not limited to those used by operating systems, software that provides security services, application and system accounts, point-of-sale (POS) terminals, payment applications, Simple Network Management Protocol (SNMP) community strings, etc.).",
				},
				{
					ID:          "2_1_1",
					Name:        "2.1.1",
					Description: "For wireless environments connected to the cardholder data environment or transmitting cardholder data, change ALL wireless vendor defaults at installation, including but not limited to default wireless encryption keys, passwords, and SNMP community strings.",
				},
				{
					ID:          "2_2",
					Name:        "2.2",
					Description: "Develop configuration standards for all system components. Assure that these standards address all known security vulnerabilities and are consistent with industry-accepted system hardening  Sources of industry-accepted system hardening standards may include, but are not limited to: • Center for Internet Security (CIS) • International Organization for Standardization (ISO) • SysAdmin Audit Network Security (SANS) Institute • National Institute of Standards Technology (NIST).",
				},
				{
					ID:          "2_2_1",
					Name:        "2.2.1",
					Description: "Implement only one primary function per server to prevent functions that require different security levels from co-existing on the same server. (For example, web servers, database servers, and DNS should be implemented on separate servers.) Note: Where virtualization technologies are in use, implement only one primary function per virtual system component.",
				},
				{
					ID:          "2_2_2",
					Name:        "2.2.2",
					Description: "Enable only necessary services, protocols, daemons, etc., as required for the function of the system.",
				},
				{
					ID:          "2_2_3",
					Name:        "2.2.3",
					Description: "Implement additional security features for any required services, protocols, or daemons that are considered to be insecure.",
				},
				{
					ID:          "2_2_4",
					Name:        "2.2.4",
					Description: "Configure system security parameters to prevent misuse.",
				},
				{
					ID:          "2_2_5",
					Name:        "2.2.5",
					Description: "Remove all unnecessary functionality, such as scripts, drivers, features, subsystems, file systems, and unnecessary web servers.",
				},
				{
					ID:          "2_3",
					Name:        "2.3",
					Description: "Encrypt all non-console administrative access using strong cryptography.",
				},
				{
					ID:          "2_4",
					Name:        "2.4",
					Description: "Maintain an inventory of system components that are in scope for PCI DSS.",
				},
				{
					ID:          "2_5",
					Name:        "2.5",
					Description: "Ensure that security policies and operational procedures for managing vendor defaults and other security parameters are documented, in use, and known to all affected parties.",
				},
				{
					ID:          "2_6",
					Name:        "2.6",
					Description: "Shared hosting providers must protect each entity’s hosted environment and cardholder data. These providers must meet specific requirements as detailed in Appendix A1: Additional PCI DSS Requirements for Shared Hosting Providers.",
				},
			},
		},
		{
			ID:          "3",
			Name:        "3",
			Description: "Protected stored cardholder data",
			Controls: []Control{
				{
					ID:          "3_1",
					Name:        "3.1",
					Description: "Keep cardholder data storage to a minimum by implementing data retention and disposal policies, procedures and processes that include at least the following for all cardholder data (CHD) storage: • Limiting data storage amount and retention time to that which is required for legal, regulatory, and/or business requirements • Specific retention requirements for cardholder data • Processes for secure deletion of data when no longer needed • A quarterly process for identifying and securely deleting stored cardholder data that exceeds defined retention.",
				},
				{
					ID:          "3_2",
					Name:        "3.2",
					Description: "Do not store sensitive authentication data after authorization (even if encrypted). If sensitive authentication data is received, render all data unrecoverable upon completion of the authorization process. It is permissible for issuers and companies that support issuing services to store sensitive authentication data if: • There is a business justification and • The data is stored securely. Sensitive authentication data includes the data as cited in the following Requirements 3.2.1 through 3.2.3:",
				},
				{
					ID:          "3_2_1",
					Name:        "3.2.1",
					Description: "Do not store the full contents of any track (from the magnetic stripe located on the back of a card, equivalent data contained on a chip, or elsewhere) after authorization. This data is alternatively called full track, track, track 1, track 2, and magnetic-stripe data.  Note: In the normal course of business, the following data elements from the magnetic stripe may need to be retained: • The cardholder’s name • Primary account number (PAN) • Expiration date • Service code To minimize risk, store only these data elements as needed for business.",
				},
				{
					ID:          "3_2_2",
					Name:        "3.2.2",
					Description: "Do not store the card verification code or value (three-digit or four-digit number printed on the front or back of a payment card used to verify card-not-present transactions) after authorization.",
				},
				{
					ID:          "3_2_3",
					Name:        "3.2.3",
					Description: "Do not store the personal identification number (PIN) or the encrypted PIN block after authorization.",
				},
				{
					ID:          "3_3",
					Name:        "3.3",
					Description: "Mask PAN when displayed (the first six and last four digits are the maximum number of digits to be displayed), such that only personnel with a legitimate business need can see more than the first six/last four digits of the PAN. Note: This requirement does not supersede stricter requirements in place for displays of cardholder data—for example, legal or payment card brand requirements for point-of-sale (POS) receipts.",
				},
				{
					ID:          "3_4",
					Name:        "3.4",
					Description: "Render PAN unreadable anywhere it is stored (including on portable digital media, backup media, and in logs) by using any of the following approaches: • One-way hashes based on strong cryptography, (hash must be of the entire PAN) • Truncation (hashing cannot be used to replace the truncated segment of PAN) • Index tokens and pads (pads must be securely stored) • Strong cryptography with associated key-management processes and procedures. Note: It is a relatively trivial effort for a malicious individual to reconstruct original PAN data if they have access to both the truncated and hashed version of a PAN. Where hashed and truncated versions of the same PAN are present in an entity’s environment, additional controls must be in place to ensure that the hashed and truncated versions cannot be correlated to reconstruct the original PAN.",
				},
				{
					ID:          "3_4_1",
					Name:        "3.4.1",
					Description: "If disk encryption is used (rather than file- or column-level database encryption), logical access must be managed separately and independently of native operating system authentication and access control mechanisms (for example, by not using local user account databases or general network login credentials). Decryption keys must not be associated with user accounts. Note: This requirement applies in addition to all other PCI DSS encryption and key-management requirements.",
				},
				{
					ID:          "3_5",
					Name:        "3.5",
					Description: "Document and implement procedures to protect keys used to secure stored cardholder data against disclosure and misuse: Note: This requirement applies to keys used to encrypt stored cardholder data, and also applies to key-encrypting keys used to protect data-encrypting keys—such key-encrypting keys must be at least as strong as the data-encrypting key.",
				},
				{
					ID:          "3_5_1",
					Name:        "3.5.1",
					Description: " Additional requirement for service providers only: Maintain a documented description of the cryptographic architecture that includes: • Details of all algorithms, protocols, and keys used for the protection of cardholder data, including key strength and expiry date • Description of the key usage for each key. • Inventory of any HSMs and other SCDs used for key management Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
				{
					ID:          "3_5_2",
					Name:        "3.5.2",
					Description: "Restrict access to cryptographic keys to the fewest number of custodians necessary.",
				},
				{
					ID:          "3_5_3",
					Name:        "3.5.3",
					Description: "Store secret and private keys used to encrypt/decrypt cardholder data in one (or more) of the following forms at all times: • Encrypted with a key-encrypting key that is at least as strong as the data-encrypting key, and that is stored separately from the data-encrypting key • Within a secure cryptographic device (such as a hardware (host) security module (HSM) or PTS-approved point-of-interaction device) • As at least two full-length key components or key shares, in accordance with an industry-accepted method Note: It is not required that public keys be stored in one of these forms.",
				},
				{
					ID:          "3_5_4",
					Name:        "3.5.4",
					Description: "Store cryptographic keys in the fewest possible locations.",
				},
				{
					ID:          "3_6",
					Name:        "3.6",
					Description: "Fully document and implement all key-management processes and procedures for cryptographic keys used for encryption of cardholder data, including the following: Note: Numerous industry standards for key management are available from various resources including NIST, which can be found at http://csrc.nist.gov.",
				},
				{
					ID:          "3_6_1",
					Name:        "3.6.1",
					Description: "Generation of strong cryptographic keys",
				},
				{
					ID:          "3_6_2",
					Name:        "3.6.2",
					Description: "Secure cryptographic key distribution",
				},
				{
					ID:          "3_6_3",
					Name:        "3.6.3",
					Description: "Secure cryptographic key storage",
				},
				{
					ID:          "3_6_4",
					Name:        "3.6.4",
					Description: "Cryptographic key changes for keys that have reached the end of their cryptoperiod (for example, after a defined period of time has passed and/or after a certain amount of cipher-text has been produced by a given key), as defined by the associated application vendor or key owner, and based on industry best practices and guidelines (for example, NIST Special Publication 800-57).",
				},
				{
					ID:          "3_6_5",
					Name:        "3.6.5",
					Description: "Retirement or replacement (for example, archiving, destruction, and/or revocation) of keys as deemed necessary when the integrity of the key has been weakened (for example, departure of an employee with knowledge of a clear-text key component), or keys are suspected of being compromised. Note: If retired or replaced cryptographic keys need to be retained, these keys must be securely archived (for example, by using a key-encryption key). Archived cryptographic keys should only be used for decryption/verification purposes.",
				},
				{
					ID:          "3_6_6",
					Name:        "3.6.6",
					Description: "If manual clear-text cryptographic key-management operations are used, these operations must be managed using split knowledge and dual control. Note: Examples of manual key-management operations include, but are not limited to: key generation, transmission, loading, storage and destruction.",
				},
				{
					ID:          "3_6_7",
					Name:        "3.6.7",
					Description: "Prevention of unauthorized substitution of cryptographic keys.",
				},
				{
					ID:          "3_6_8",
					Name:        "3.6.8",
					Description: "Requirement for cryptographic key custodians to formally acknowledge that they understand and accept their key-custodian responsibilities.",
				},
				{
					ID:          "3_7",
					Name:        "3.7",
					Description: "Ensure that security policies and operational procedures for protecting stored cardholder data are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "4",
			Name:        "4",
			Description: "Encrypt transmission of cardholder data across open, public networks",
			Controls: []Control{
				{
					ID:          "4_1",
					Name:        "4.1",
					Description: "Use strong cryptography and security protocols to safeguard sensitive cardholder data during transmission over open, public networks, including the following: • Only trusted keys and certificates are accepted. • The protocol in use only supports secure versions or configurations. • The encryption strength is appropriate for the encryption methodology in use.  Examples of open, public networks include but are not limited to: • The Internet • Wireless technologies, including 802.11 and Bluetooth • Cellular technologies, for example, Global System for Mobile communications (GSM), Code division multiple access (CDMA) • General Packet Radio Service (GPRS). • Satellite communications.",
				},
				{
					ID:          "4_1_1",
					Name:        "4.1.1",
					Description: "Ensure wireless networks transmitting cardholder data or connected to the cardholder data environment, use industry best practices to implement strong encryption for authentication and transmission.",
				},
				{
					ID:          "4_2",
					Name:        "4.2",
					Description: "Never send unprotected PANs by end-user messaging technologies (for example, e-mail, instant messaging, SMS, chat, etc.).",
				},
				{
					ID:          "4_3",
					Name:        "4.3",
					Description: "Ensure that security policies and operational procedures for encrypting transmissions of cardholder data are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "5",
			Name:        "5",
			Description: "Protect all systems against malware and regularly update anti-virus software or programs",
			Controls: []Control{
				{
					ID:          "5_1",
					Name:        "5.1",
					Description: "Deploy anti-virus software on all systems commonly affected by malicious software (particularly personal computers and servers).",
				},
				{
					ID:          "5_1_1",
					Name:        "5.1.1",
					Description: "Ensure that anti-virus programs are capable of detecting, removing, and protecting against all known types of malicious software.",
				},
				{
					ID:          "5_1_2",
					Name:        "5.1.2",
					Description: "For systems considered to be not commonly affected by malicious software, perform periodic evaluations to identify and evaluate evolving malware threats in order to confirm whether such systems continue to not require anti-virus software.",
				},
				{
					ID:          "5_2",
					Name:        "5.2",
					Description: "Ensure that all anti-virus mechanisms are maintained as follows: • Are kept current, • Perform periodic scans • Generate audit logs which are retained per PCI DSS Requirement 10.7.",
				},
				{
					ID:          "5_3",
					Name:        "5.3",
					Description: "Ensure that anti-virus mechanisms are actively running and cannot be disabled or altered by users, unless specifically authorized by management on a case-by-case basis for a limited time period. Note: Anti-virus solutions may be temporarily disabled only if there is legitimate technical need, as authorized by management on a case-by-case basis. If anti-virus protection needs to be disabled for a specific purpose, it must be formally authorized. Additional security measures may also need to be implemented for the period of time during which anti-virus protection is not active.",
				},
				{
					ID:          "5_4",
					Name:        "5.4",
					Description: "Ensure that security policies and operational procedures for protecting systems against malware are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "6",
			Name:        "6",
			Description: "Develop and maintain secure systems and applications",
			Controls: []Control{
				{
					ID:          "6_1",
					Name:        "6.1",
					Description: "Establish a process to identify security vulnerabilities, using reputable outside sources for security vulnerability information, and assign a risk ranking (for example, as “high,” “medium,” or “low”) to newly discovered security vulnerabilities. Note: Risk rankings should be based on industry best practices as well as consideration of potential impact. For example, criteria for ranking vulnerabilities may include consideration of the CVSS base score, and/or the classification by the vendor, and/or type of systems affected. Methods for evaluating vulnerabilities and assigning risk ratings will vary based on an organization’s environment and risk-assessment strategy. Risk rankings should, at a minimum, identify all vulnerabilities considered to be a “high risk” to the environment. In addition to the risk ranking, vulnerabilities may be considered “critical” if they pose an imminent threat to the environment, impact critical systems, and/or would result in a potential compromise if not addressed. Examples of critical systems may include security systems, public-facing devices and systems, databases, and other systems that store, process, or transmit cardholder data.",
				},
				{
					ID:          "6_2",
					Name:        "6.2",
					Description: "Ensure that all system components and software are protected from known vulnerabilities by installing applicable vendor-supplied security patches. Install critical security patches within one month of release. Note: Critical security patches should be identified according to the risk ranking process defined in Requirement 6.1.",
				},
				{
					ID:          "6_3",
					Name:        "6.3",
					Description: "Develop internal and external software applications (including web-based administrative access to applications) securely, as follows: • In accordance with PCI DSS (for example, secure authentication and logging) • Based on industry standards and/or best practices. • Incorporating information security throughout the software-development life cycle  Note: This applies to all software developed internally as well as bespoke or custom software developed by a third party.",
				},
				{
					ID:          "6_3_1",
					Name:        "6.3.1",
					Description: "Remove development, test and/or custom application accounts, user IDs, and passwords before applications become active or are released to customers.",
				},
				{
					ID:          "6_3_2",
					Name:        "6.3.2",
					Description: "Review custom code prior to release to production or customers in order to identify any potential coding vulnerability (using either manual or automated processes) to include at least the following: • Code changes are reviewed by individuals other than the originating code author, and by individuals knowledgeable about code-review techniques and secure coding practices. • Code reviews ensure code is developed according to secure coding guidelines • Appropriate corrections are implemented prior to release. • Code-review results are reviewed and approved by management prior to release. Note: This requirement for code reviews applies to all custom code (both internal and public-facing), as part of the system development life cycle. Code reviews can be conducted by knowledgeable internal personnel or third parties. Public-facing web applications are also subject to additional controls, to address ongoing threats and vulnerabilities after implementation, as defined at PCI DSS Requirement 6.6.",
				},
				{
					ID:          "6_4",
					Name:        "6.4",
					Description: "Follow change control processes and procedures for all changes to system components. The processes must include the following:",
				},
				{
					ID:          "6_4_1",
					Name:        "6.4.1",
					Description: "Separate development/test environments from production environments, and enforce the separation with access controls.",
				},
				{
					ID:          "6_4_2",
					Name:        "6.4.2",
					Description: "Separation of duties between development/test and production environments",
				},
				{
					ID:          "6_4_3",
					Name:        "6.4.3",
					Description: "Production data (live PANs) are not used for testing or development",
				},
				{
					ID:          "6_4_4",
					Name:        "6.4.4",
					Description: "Removal of test data and accounts from system components before the system becomes active/goes into production.",
				},
				{
					ID:          "6_4_5",
					Name:        "6.4.5",
					Description: "Change control procedures must include the following:",
				},
				{
					ID:          "6_4_5_1",
					Name:        "6.4.5.1",
					Description: "Documentation of impact.",
				},
				{
					ID:          "6_4_5_2",
					Name:        "6.4.5.2",
					Description: "Documented change approval by authorized parties.",
				},
				{
					ID:          "6_4_5_3",
					Name:        "6.4.5.3",
					Description: "Functionality testing to verify that the change does not adversely impact the security of the system.",
				},
				{
					ID:          "6_4_5_4",
					Name:        "6.4.5.4",
					Description: "Back-out procedures.",
				},
				{
					ID:          "6_4_6",
					Name:        "6.4.6",
					Description: "Upon completion of a significant change, all relevant PCI DSS requirements must be implemented on all new or changed systems and networks, and documentation updated as applicable. Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
				{
					ID:          "6_5",
					Name:        "6.5",
					Description: "Address common coding vulnerabilities in software-development processes as follows: • Train developers at least annually in up-to-date secure coding techniques, including how to avoid common coding vulnerabilities. • Develop applications based on secure coding guidelines. Note: The vulnerabilities listed at 6.5.1 through 6.5.10 were current with industry best practices when this version of PCI DSS was published. However, as industry best practices for vulnerability management are updated (for example, the OWASP Guide, SANS CWE Top 25, CERT Secure Coding, etc.), the current best practices must be used for these requirements.",
				},
				{
					ID:          "6_5_1",
					Name:        "6.5.1",
					Description: "Injection flaws, particularly SQL injection. Also consider OS Command Injection, LDAP and XPath injection flaws as well as other injection flaws.",
				},
				{
					ID:          "6_5_2",
					Name:        "6.5.2",
					Description: "Buffer overflows",
				},
				{
					ID:          "6_5_3",
					Name:        "6.5.3",
					Description: "Insecure cryptographic storage",
				},
				{
					ID:          "6_5_4",
					Name:        "6.5.4",
					Description: "Insecure communications",
				},
				{
					ID:          "6_5_5",
					Name:        "6.5.5",
					Description: "Improper error handling",
				},
				{
					ID:          "6_5_6",
					Name:        "6.5.6",
					Description: "All “high risk” vulnerabilities identified in the vulnerability identification process (as defined in PCI DSS Requirement 6.1).",
				},
				{
					ID:          "6_5_7",
					Name:        "6.5.7",
					Description: "Cross-site scripting (XSS)",
				},
				{
					ID:          "6_5_8",
					Name:        "6.5.8",
					Description: "Improper access control (such as insecure direct object references, failure to restrict URL access, directory traversal, and failure to restrict user access to functions).",
				},
				{
					ID:          "6_5_9",
					Name:        "6.5.9",
					Description: "Cross-site request forgery (CSRF)",
				},
				{
					ID:          "6_5_10",
					Name:        "6.5.10",
					Description: "Broken authentication and session management",
				},
				{
					ID:          "6_6",
					Name:        "6.6",
					Description: "For public-facing web applications, address new threats and vulnerabilities on an ongoing basis and ensure these applications are protected against known attacks by either of the following methods: • Reviewing public-facing web applications via manual or automated application vulnerability security assessment tools or methods, at least annually and after any changes Note: This assessment is not the same as the vulnerability scans performed for Requirement 11.2.  • Installing an automated technical solution that detects and prevents web-based attacks (for example, a web-application firewall) in front of public-facing web applications, to continually check all traffic.",
				},
				{
					ID:          "6_7",
					Name:        "6.7",
					Description: "Ensure that security policies and operational procedures for developing and maintaining secure systems and applications are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "7",
			Name:        "7",
			Description: "Restrict access to cardholder data by business need to know",
			Controls: []Control{
				{
					ID:          "7_1",
					Name:        "7.1",
					Description: "Limit access to system components and cardholder data to only those individuals whose job requires such access.",
				},
				{
					ID:          "7_1_1",
					Name:        "7.1.1",
					Description: "Define access needs for each role, including: • System components and data resources that each role needs to access for their job function • Level of privilege required (for example, user, administrator, etc.) for accessing resources.",
				},
				{
					ID:          "7_1_2",
					Name:        "7.1.2",
					Description: "Restrict access to privileged user IDs to least privileges necessary to perform job responsibilities.",
				},
				{
					ID:          "7_1_3",
					Name:        "7.1.3",
					Description: "Assign access based on individual personnel’s job classification and function.",
				},
				{
					ID:          "7_1_4",
					Name:        "7.1.4",
					Description: "Require documented approval by authorized parties specifying required privileges.",
				},
				{
					ID:          "7_2",
					Name:        "7.2",
					Description: "Establish an access control system(s) for systems components that restricts access based on a user’s need to know, and is set to “deny all” unless specifically allowed. This access control system(s) must include the following:",
				},
				{
					ID:          "7_2_1",
					Name:        "7.2.1",
					Description: "Coverage of all system components",
				},
				{
					ID:          "7_2_2",
					Name:        "7.2.2",
					Description: "Assignment of privileges to individuals based on job classification and function.",
				},
				{
					ID:          "7_2_3",
					Name:        "7.2.3",
					Description: "Default “deny-all” setting.",
				},
				{
					ID:          "7_3",
					Name:        "7.3",
					Description: "Ensure that security policies and operational procedures for restricting access to cardholder data are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "8",
			Name:        "8",
			Description: "Identify and authenticate access to system components",
			Controls: []Control{
				{
					ID:          "8_1",
					Name:        "8.1",
					Description: "Define and implement policies and procedures to ensure proper user identification management for non-consumer users and administrators on all system components as follows:",
				},
				{
					ID:          "8_1_1",
					Name:        "8.1.1",
					Description: "Assign all users a unique ID before allowing them to access system components or cardholder data.",
				},
				{
					ID:          "8_1_2",
					Name:        "8.1.2",
					Description: "Control addition, deletion, and modification of user IDs, credentials, and other identifier objects.",
				},
				{
					ID:          "8_1_3",
					Name:        "8.1.3",
					Description: "Immediately revoke access for any terminated users.",
				},
				{
					ID:          "8_1_4",
					Name:        "8.1.4",
					Description: "Remove/disable inactive user accounts within 90 days.",
				},
				{
					ID:          "8_1_5",
					Name:        "8.1.5",
					Description: "Manage IDs used by third parties to access, support, or maintain system components via remote access as follows: • Enabled only during the time period needed and disabled when not in use. • Monitored when in use.",
				},
				{
					ID:          "8_1_6",
					Name:        "8.1.6",
					Description: "Limit repeated access attempts by locking out the user ID after not more than six attempts.",
				},
				{
					ID:          "8_1_7",
					Name:        "8.1.7",
					Description: "Set the lockout duration to a minimum of 30 minutes or until an administrator enables the user ID.",
				},
				{
					ID:          "8_1_8",
					Name:        "8.1.8",
					Description: "If a session has been idle for more than 15 minutes, require the user to re-authenticate to re-activate the terminal or session.",
				},
				{
					ID:          "8_2",
					Name:        "8.2",
					Description: "In addition to assigning a unique ID, ensure proper user-authentication management for non-consumer users and administrators on all system components by employing at least one of the following methods to authenticate all users: • Something you know, such as a password or passphrase • Something you have, such as a token device or smart card • Something you are, such as a biometric.",
				},
				{
					ID:          "8_2_1",
					Name:        "8.2.1",
					Description: "Using strong cryptography, render all authentication credentials (such as passwords/phrases) unreadable during transmission and storage on all system components.",
				},
				{
					ID:          "8_2_2",
					Name:        "8.2.2",
					Description: "Verify user identity before modifying any authentication credential—for example, performing password resets, provisioning new tokens, or generating new keys.",
				},
				{
					ID:          "8_2_3",
					Name:        "8.2.3",
					Description: "Passwords/passphrases must meet the following: • Require a minimum length of at least seven characters. • Contain both numeric and alphabetic characters. Alternatively, the passwords/passphrases must have complexity and strength at least equivalent to the parameters specified above.",
				},
				{
					ID:          "8_2_4",
					Name:        "8.2.4",
					Description: "Change user passwords/passphrases at least once every 90 days.",
				},
				{
					ID:          "8_2_5",
					Name:        "8.2.5",
					Description: "Do not allow an individual to submit a new password/passphrase that is the same as any of the last four passwords/passphrases he or she has used.",
				},
				{
					ID:          "8_2_6",
					Name:        "8.2.6",
					Description: "Set passwords/passphrases for first-time use and upon reset to a unique value for each user, and change immediately after the first use.",
				},
				{
					ID:          "8_3",
					Name:        "8.3",
					Description: "Secure all individual non-console administrative access and all remote access to the CDE using multi-factor authentication. Note: Multi-factor authentication requires that a minimum of two of the three authentication methods (see Requirement 8.2 for descriptions of authentication methods) be used for authentication. Using one factor twice (for example, using two separate passwords) is not considered multi-factor authentication.",
				},
				{
					ID:          "8_3_1",
					Name:        "8.3.1",
					Description: "Incorporate multi-factor authentication for all non-console access into the CDE for personnel with administrative access.   Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
				{
					ID:          "8_3_2",
					Name:        "8.3.2",
					Description: "Incorporate multi-factor authentication for all remote network access (both user and administrator, and including third party access for support or maintenance) originating from outside the entity's network.",
				},
				{
					ID:          "8_4",
					Name:        "8.4",
					Description: "Document and communicate authentication policies and procedures to all users including: • Guidance on selecting strong authentication credentials • Guidance for how users should protect their authentication credentials • Instructions not to reuse previously used passwords • Instructions to change passwords if there is any suspicion the password could be compromised.",
				},
				{
					ID:          "8_5",
					Name:        "8.5",
					Description: "Do not use group, shared, or generic IDs, passwords, or other authentication methods as follows: • Generic user IDs are disabled or removed. • Shared user IDs do not exist for system administration and other critical functions. • Shared and generic user IDs are not used to administer any system components.",
				},
				{
					ID:          "8_5_1",
					Name:        "8.5.1",
					Description: "Additional requirement for service providers only: Service providers with remote access to customer premises (for example, for support of POS systems or servers) must use a unique authentication credential (such as a password/phrase) for each customer. Note: This requirement is not intended to apply to shared hosting providers accessing their own hosting environment, where multiple customer environments are hosted.",
				},
				{
					ID:          "8_6",
					Name:        "8.6",
					Description: "Where other authentication mechanisms are used (for example, physical or logical security tokens, smart cards, certificates, etc.), use of these mechanisms must be assigned as follows: • Authentication mechanisms must be assigned to an individual account and not shared among multiple accounts. • Physical and/or logical controls must be in place to ensure only the intended account can use that mechanism to gain access.",
				},
				{
					ID:          "8_7",
					Name:        "8.7",
					Description: "All access to any database containing cardholder data (including access by applications, administrators, and all other users) is restricted as follows: • All user access to, user queries of, and user actions on databases are through programmatic methods. • Only database administrators have the ability to directly access or query databases. • Application IDs for database applications can only be used by the applications (and not by individual users or other non-application processes).",
				},
				{
					ID:          "8_8",
					Name:        "8.8",
					Description: "Ensure that security policies and operational procedures for identification and authentication are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "9",
			Name:        "9",
			Description: "Restrict physical access to cardholder data",
			Controls: []Control{
				{
					ID:          "9_1",
					Name:        "9.1",
					Description: "Use appropriate facility entry controls to limit and monitor physical access to systems in the cardholder data environment.",
				},
				{
					ID:          "9_1_1",
					Name:        "9.1.1",
					Description: "Use either video cameras or access control mechanisms (or both) to monitor individual physical access to sensitive areas. Review collected data and correlate with other entries. Store for at least three months, unless otherwise restricted by law. Note: “Sensitive areas” refers to any data center, server room or any area that houses systems that store, process, or transmit cardholder data. This excludes public-facing areas where only point-of-sale terminals are present, such as the cashier areas in a retail store.",
				},
				{
					ID:          "9_1_2",
					Name:        "9.1.2",
					Description: "Implement physical and/or logical controls to restrict access to publicly accessible network jacks. For example, network jacks located in public areas and areas accessible to visitors could be disabled and only enabled when network access is explicitly authorized. Alternatively, processes could be implemented to ensure that visitors are escorted at all times in areas with active network jacks.",
				},
				{
					ID:          "9_1_3",
					Name:        "9.1.3",
					Description: "Restrict physical access to wireless access points, gateways, handheld devices, networking/communications hardware, and telecommunication lines.",
				},
				{
					ID:          "9_2",
					Name:        "9.2",
					Description: "Develop procedures to easily distinguish between onsite personnel and visitors, to include: • Identifying onsite personnel and visitors (for example, assigning badges) • Changes to access requirements • Revoking or terminating onsite personnel and expired visitor identification (such as ID badges).",
				},
				{
					ID:          "9_3",
					Name:        "9.3",
					Description: "Control physical access for onsite personnel to sensitive areas as follows: • Access must be authorized and based on individual job function. • Access is revoked immediately upon termination, and all physical access mechanisms, such as keys, access cards, etc., are returned or disabled.",
				},
				{
					ID:          "9_4",
					Name:        "9.4",
					Description: "Implement procedures to identify and authorize visitors. Procedures should include the following:",
				},
				{
					ID:          "9_4_1",
					Name:        "9.4.1",
					Description: "Visitors are authorized before entering, and escorted at all times within, areas where cardholder data is processed or maintained.",
				},
				{
					ID:          "9_4_2",
					Name:        "9.4.2",
					Description: "Visitors are identified and given a badge or other identification that expires and that visibly distinguishes the visitors from onsite personnel.",
				},
				{
					ID:          "9_4_3",
					Name:        "9.4.3",
					Description: "Visitors are asked to surrender the badge or identification before leaving the facility or at the date of expiration.",
				},
				{
					ID:          "9_4_4",
					Name:        "9.4.4",
					Description: "A visitor log is used to maintain a physical audit trail of visitor activity to the facility as well as computer rooms and data centers where cardholder data is stored or transmitted. Document the visitor’s name, the firm represented, and the onsite personnel authorizing physical access on the log. Retain this log for a minimum of three months, unless otherwise restricted by law.",
				},
				{
					ID:          "9_5",
					Name:        "9.5",
					Description: "Physically secure all media.",
				},
				{
					ID:          "9_5_1",
					Name:        "9.5.1",
					Description: "Store media backups in a secure location, preferably an off-site facility, such as an alternate or backup site, or a commercial storage facility. Review the location’s security at least annually.",
				},
				{
					ID:          "9_6",
					Name:        "9.6",
					Description: "Maintain strict control over the internal or external distribution of any kind of media, including the following:",
				},
				{
					ID:          "9_6_1",
					Name:        "9.6.1",
					Description: "Classify media so the sensitivity of the data can be determined.",
				},
				{
					ID:          "9_6_2",
					Name:        "9.6.2",
					Description: "Send the media by secured courier or other delivery method that can be accurately tracked.",
				},
				{
					ID:          "9_6_3",
					Name:        "9.6.3",
					Description: "Ensure management approves any and all media that is moved from a secured area (including when media is distributed to individuals).",
				},
				{
					ID:          "9_7",
					Name:        "9.7",
					Description: "Maintain strict control over the storage and accessibility of media.",
				},
				{
					ID:          "9_7_1",
					Name:        "9.7.1",
					Description: "Properly maintain inventory logs of all media and conduct media inventories at least annually.",
				},
				{
					ID:          "9_8",
					Name:        "9.8",
					Description: "Destroy media when it is no longer needed for business or legal reasons as follows:",
				},
				{
					ID:          "9_8_1",
					Name:        "9.8.1",
					Description: "Shred, incinerate, or pulp hard-copy materials so that cardholder data cannot be reconstructed. Secure storage containers used for materials that are to be destroyed.",
				},
				{
					ID:          "9_8_2",
					Name:        "9.8.2",
					Description: "Render cardholder data on electronic media unrecoverable so that cardholder data cannot be reconstructed.",
				},
				{
					ID:          "9_9",
					Name:        "9.9",
					Description: "Protect devices that capture payment card data via direct physical interaction with the card from tampering and substitution. Note: These requirements apply to card-reading devices used in card-present transactions (that is, card swipe or dip) at the point of sale. This requirement is not intended to apply to manual key-entry components such as computer keyboards and POS keypads.",
				},
				{
					ID:          "9_9_1",
					Name:        "9.9.1",
					Description: "Maintain an up-to-date list of devices. The list should include the following: • Make, model of device • Location of device (for example, the address of the site or facility where the device is located) • Device serial number or other method of unique identification.",
				},
				{
					ID:          "9_9_2",
					Name:        "9.9.2",
					Description: "Periodically inspect device surfaces to detect tampering (for example, addition of card skimmers to devices), or substitution (for example, by checking the serial number or other device characteristics to verify it has not been swapped with a fraudulent device). Note: Examples of signs that a device might have been tampered with or substituted include unexpected attachments or cables plugged into the device, missing or changed security labels, broken or differently colored casing, or changes to the serial number or other external markings.",
				},
				{
					ID:          "9_9_3",
					Name:        "9.9.3",
					Description: "Provide training for personnel to be aware of attempted tampering or replacement of devices. Training should include the following: • Verify the identity of any third-party persons claiming to be repair or maintenance personnel, prior to granting them access to modify or troubleshoot devices. • Do not install, replace, or return devices without verification. • Be aware of suspicious behavior around devices (for example, attempts by unknown persons to unplug or open devices). • Report suspicious behavior and indications of device tampering or substitution to appropriate personnel (for example, to a manager or security officer).",
				},
				{
					ID:          "9_10",
					Name:        "9.10",
					Description: "Ensure that security policies and operational procedures for restricting physical access to cardholder data are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "10",
			Name:        "10",
			Description: "Track and monitor all access to network resources and cardholder data",
			Controls: []Control{
				{
					ID:          "10_1",
					Name:        "10.1",
					Description: "Implement audit trails to link all access to system components to each individual user.",
				},
				{
					ID:          "10_2",
					Name:        "10.2",
					Description: "Implement automated audit trails for all system components to reconstruct the following events:",
				},
				{
					ID:          "10_2_1",
					Name:        "10.2.1",
					Description: "All individual user accesses to cardholder data",
				},
				{
					ID:          "10_2_2",
					Name:        "10.2.2",
					Description: "All actions taken by any individual with root or administrative privileges",
				},
				{
					ID:          "10_2_3",
					Name:        "10.2.3",
					Description: "Access to all audit trails",
				},
				{
					ID:          "10_2_4",
					Name:        "10.2.4",
					Description: "Invalid logical access attempts",
				},
				{
					ID:          "10_2_5",
					Name:        "10.2.5",
					Description: "Use of and changes to identification and authentication mechanisms—including but not limited to creation of new accounts and elevation of privileges—and all changes, additions, or deletions to accounts with root or administrative privileges",
				},
				{
					ID:          "10_2_6",
					Name:        "10.2.6",
					Description: "Initialization, stopping, or pausing of the audit logs",
				},
				{
					ID:          "10_2_7",
					Name:        "10.2.7",
					Description: "Creation and deletion of system-level objects",
				},
				{
					ID:          "10_3",
					Name:        "10.3",
					Description: "Record at least the following audit trail entries for all system components for each event:",
				},
				{
					ID:          "10_3_1",
					Name:        "10.3.1",
					Description: "User identification",
				},
				{
					ID:          "10_3_2",
					Name:        "10.3.2",
					Description: "Type of event",
				},
				{
					ID:          "10_3_3",
					Name:        "10.3.3",
					Description: "Date and time",
				},
				{
					ID:          "10_3_4",
					Name:        "10.3.4",
					Description: "Success or failure indication",
				},
				{
					ID:          "10_3_5",
					Name:        "10.3.5",
					Description: "Origination of event",
				},
				{
					ID:          "10_3_6",
					Name:        "10.3.6",
					Description: "Identity or name of affected data, system component, or resource.",
				},
				{
					ID:          "10_4",
					Name:        "10.4",
					Description: "Using time-synchronization technology, synchronize all critical system clocks and times and ensure that the following is implemented for acquiring, distributing, and storing time. Note: One example of time synchronization technology is Network Time Protocol (NTP).",
				},
				{
					ID:          "10_4_1",
					Name:        "10.4.1",
					Description: "Critical systems have the correct and consistent time.",
				},
				{
					ID:          "10_4_2",
					Name:        "10.4.2",
					Description: "Time data is protected.",
				},
				{
					ID:          "10_4_3",
					Name:        "10.4.3",
					Description: "Time settings are received from industry-accepted time sources.",
				},
				{
					ID:          "10_5",
					Name:        "10.5",
					Description: "Secure audit trails so they cannot be altered.",
				},
				{
					ID:          "10_5_1",
					Name:        "10.5.1",
					Description: "Limit viewing of audit trails to those with a job-related need.",
				},
				{
					ID:          "10_5_2",
					Name:        "10.5.2",
					Description: "Protect audit trail files from unauthorized modifications.",
				},
				{
					ID:          "10_5_3",
					Name:        "10.5.3",
					Description: "Promptly back up audit trail files to a centralized log server or media that is difficult to alter.",
				},
				{
					ID:          "10_5_4",
					Name:        "10.5.4",
					Description: "Write logs for external-facing technologies onto a secure, centralized, internal log server or media device.",
				},
				{
					ID:          "10_5_5",
					Name:        "10.5.5",
					Description: "Use file-integrity monitoring or change-detection software on logs to ensure that existing log data cannot be changed without generating alerts (although new data being added should not cause an alert).",
				},
				{
					ID:          "10_6",
					Name:        "10.6",
					Description: "Review logs and security events for all system components to identify anomalies or suspicious activity.  Note: Log harvesting, parsing, and alerting tools may be used to meet this Requirement.",
				},
				{
					ID:          "10_6_1",
					Name:        "10.6.1",
					Description: "Review the following at least daily: • All security events • Logs of all system components that store, process, or transmit CHD and/or SAD • Logs of all critical system components • Logs of all servers and system components that perform security functions (for example, firewalls, intrusion-detection systems/intrusion-prevention systems (IDS/IPS), authentication servers, e-commerce redirection servers, etc.).",
				},
				{
					ID:          "10_6_2",
					Name:        "10.6.2",
					Description: "Review logs of all other system components periodically based on the organization’s policies and risk management strategy, as determined by the organization’s annual risk assessment.",
				},
				{
					ID:          "10_6_3",
					Name:        "10.6.3",
					Description: "Follow up exceptions and anomalies identified during the review process.",
				},
				{
					ID:          "10_7",
					Name:        "10.7",
					Description: "Retain audit trail history for at least one year, with a minimum of three months immediately available for analysis (for example, online, archived, or restorable from backup).",
				},
				{
					ID:          "10_8",
					Name:        "10.8",
					Description: "Additional requirement for service providers only: Implement a process for the timely detection and reporting of failures of critical security control systems, including but not limited to failure of: • Firewalls • IDS/IPS • FIM • Anti-virus • Physical access controls • Logical access controls • Audit logging mechanisms • Segmentation controls (if used)  Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
				{
					ID:          "10_8_1",
					Name:        "10.8.1",
					Description: "Additional requirement for service providers only: Respond to failures of any critical security controls in a timely manner. Processes for responding to failures in security controls must include: • Restoring security functions • Identifying and documenting the duration (date and time start to end) of the security failure • Identifying and documenting cause(s) of failure, including root cause, and documenting remediation required to address root cause • Identifying and addressing any security issues that arose during the failure • Performing a risk assessment to determine whether further actions are required as a result of the security failure • Implementing controls to prevent cause of failure from reoccurring • Resuming monitoring of security controls  Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
				{
					ID:          "10_9",
					Name:        "10.9",
					Description: "Ensure that security policies and operational procedures for monitoring all access to network resources and cardholder data are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "11",
			Name:        "11",
			Description: "Regularly test security systems and processes",
			Controls: []Control{
				{
					ID:          "11_1",
					Name:        "11.1",
					Description: "Implement processes to test for the presence of wireless access points (802.11), and detect and identify all authorized and unauthorized wireless access points on a quarterly basis.  Note: Methods that may be used in the process include but are not limited to wireless network scans, physical/logical inspections of system components and infrastructure, network access control (NAC), or wireless IDS/IPS. Whichever methods are used, they must be sufficient to detect and identify both authorized and unauthorized devices.",
				},
				{
					ID:          "11_1_1",
					Name:        "11.1.1",
					Description: "Maintain an inventory of authorized wireless access points including a documented business justification.",
				},
				{
					ID:          "11_1_2",
					Name:        "11.1.2",
					Description: "Implement incident response procedures in the event unauthorized wireless access points are detected.",
				},
				{
					ID:          "11_2",
					Name:        "11.2",
					Description: "Run internal and external network vulnerability scans at least quarterly and after any significant change in the network (such as new system component installations, changes in network topology, firewall rule modifications, product upgrades).  Note: Multiple scan reports can be combined for the quarterly scan process to show that all systems were scanned and all applicable vulnerabilities have been addressed. Additional documentation may be required to verify non-remediated vulnerabilities are in the process of being addressed. For initial PCI DSS compliance, it is not required that four quarters of passing scans be completed if the assessor verifies 1) the most recent scan result was a passing scan, 2) the entity has documented policies and procedures requiring quarterly scanning, and 3) vulnerabilities noted in the scan results have been corrected as shown in a re-scan(s). For subsequent years after the initial PCI DSS review, four quarters of passing scans must have occurred.",
				},
				{
					ID:          "11_2_1",
					Name:        "11.2.1",
					Description: "Perform quarterly internal vulnerability scans. Address vulnerabilities and perform rescans to verify all “high risk” vulnerabilities are resolved in accordance with the entity’s vulnerability ranking (per Requirement 6.1). Scans must be performed by qualified personnel.",
				},
				{
					ID:          "11_2_2",
					Name:        "11.2.2",
					Description: "Perform quarterly external vulnerability scans, via an Approved Scanning Vendor (ASV) approved by the Payment Card Industry Security Standards Council (PCI SSC). Perform rescans as needed, until passing scans are achieved. Note: Quarterly external vulnerability scans must be performed by an Approved Scanning Vendor (ASV), approved by the Payment Card Industry Security Standards Council (PCI SSC). Refer to the ASV Program Guide published on the PCI SSC website for scan customer responsibilities, scan preparation, etc.",
				},
				{
					ID:          "11_2_3",
					Name:        "11.2.3",
					Description: "Perform internal and external scans, and rescans as needed, after any significant change. Scans must be performed by qualified personnel.",
				},
				{
					ID:          "11_3",
					Name:        "11.3",
					Description: "Implement a methodology for penetration testing that includes the following: • Is based on industry-accepted penetration testing approaches (for example, NIST SP800-115) • Includes coverage for the entire CDE perimeter and critical systems • Includes testing from both inside and outside the network • Includes testing to validate any segmentation and scope-reduction controls • Defines application-layer penetration tests to include, at a minimum, the vulnerabilities listed in Requirement 6.5 • Defines network-layer penetration tests to include components that support network functions as well as operating systems • Includes review and consideration of threats and vulnerabilities experienced in the last 12 months • Specifies retention of penetration testing results and remediation activities results.",
				},
				{
					ID:          "11_3_1",
					Name:        "11.3.1",
					Description: "Perform external penetration testing at least annually and after any significant infrastructure or application upgrade or modification (such as an operating system upgrade, a sub-network added to the environment, or a web server added to the environment).",
				},
				{
					ID:          "11_3_2",
					Name:        "11.3.2",
					Description: "Perform internal penetration testing at least annually and after any significant infrastructure or application upgrade or modification (such as an operating system upgrade, a sub-network added to the environment, or a web server added to the environment).",
				},
				{
					ID:          "11_3_3",
					Name:        "11.3.3",
					Description: "Exploitable vulnerabilities found during penetration testing are corrected and testing is repeated to verify the corrections.",
				},
				{
					ID:          "11_3_4",
					Name:        "11.3.4",
					Description: "If segmentation is used to isolate the CDE from other networks, perform penetration tests at least annually and after any changes to segmentation controls/methods to verify that the segmentation methods are operational and effective, and isolate all out-of-scope systems from systems in the CDE.",
				},
				{
					ID:          "11_3_4_1",
					Name:        "11.3.4.1",
					Description: "Additional requirement for service providers only: If segmentation is used, confirm PCI DSS scope by performing penetration testing on segmentation controls at least every six months and after any changes to segmentation controls/methods.",
				},
				{
					ID:          "11_4",
					Name:        "11.4",
					Description: "Use intrusion-detection and/or intrusion-prevention techniques to detect and/or prevent intrusions into the network. Monitor all traffic at the perimeter of the cardholder data environment as well as at critical points in the cardholder data environment, and alert personnel to suspected compromises. Keep all intrusion-detection and prevention engines, baselines, and signatures up to date.",
				},
				{
					ID:          "11_5",
					Name:        "11.5",
					Description: "Deploy a change-detection mechanism (for example, file-integrity monitoring tools) to alert personnel to unauthorized modification (including changes, additions, and deletions) of critical system files, configuration files, or content files; and configure the software to perform critical file comparisons at least weekly. Note: For change-detection purposes, critical files are usually those that do not regularly change, but the modification of which could indicate a system compromise or risk of compromise. Change-detection mechanisms such as file-integrity monitoring products usually come pre-configured with critical files for the related operating system. Other critical files, such as those for custom applications, must be evaluated and defined by the entity (that is, the merchant or service provider).",
				},
				{
					ID:          "11_5_1",
					Name:        "11.5.1",
					Description: "Implement a process to respond to any alerts generated by the change-detection solution.",
				},
				{
					ID:          "11_6",
					Name:        "11.6",
					Description: "Ensure that security policies and operational procedures for security monitoring and testing are documented, in use, and known to all affected parties.",
				},
			},
		},
		{
			ID:          "12",
			Name:        "12",
			Description: "Maintain a policy that addresses information security for all personnel",
			Controls: []Control{
				{
					ID:          "12_1",
					Name:        "12.1",
					Description: "Establish, publish, maintain, and disseminate a security policy.",
				},
				{
					ID:          "12_1_1",
					Name:        "12.1.1",
					Description: "Review the security policy at least annually and update the policy when the environment changes.",
				},
				{
					ID:          "12_2",
					Name:        "12.2",
					Description: "Implement a risk-assessment process that: • Is performed at least annually and upon significant changes to the environment (for example, acquisition, merger, relocation, etc.), • Identifies critical assets, threats, and vulnerabilities, and • Results in a formal, documented analysis of risk. Examples of risk-assessment methodologies include but are not limited to OCTAVE, ISO 27005 and NIST SP 800-30.",
				},
				{
					ID:          "12_3",
					Name:        "12.3",
					Description: "Develop usage policies for critical technologies and define proper use of these technologies. Note: Examples of critical technologies include, but are not limited to, remote access and wireless technologies, laptops, tablets, removable electronic media, e-mail usage and Internet usage. Ensure these usage policies require the following:",
				},
				{
					ID:          "12_3_1",
					Name:        "12.3.1",
					Description: "Explicit approval by authorized parties",
				},
				{
					ID:          "12_3_2",
					Name:        "12.3.2",
					Description: "Authentication for use of the technology",
				},
				{
					ID:          "12_3_3",
					Name:        "12.3.3",
					Description: "A list of all such devices and personnel with access",
				},
				{
					ID:          "12_3_4",
					Name:        "12.3.4",
					Description: "A method to accurately and readily determine owner, contact information, and purpose (for example, labeling, coding, and/or inventorying of devices)",
				},
				{
					ID:          "12_3_5",
					Name:        "12.3.5",
					Description: "Acceptable uses of the technology",
				},
				{
					ID:          "12_3_6",
					Name:        "12.3.6",
					Description: "Acceptable network locations for the technologies",
				},
				{
					ID:          "12_3_7",
					Name:        "12.3.7",
					Description: "List of company-approved products",
				},
				{
					ID:          "12_3_8",
					Name:        "12.3.8",
					Description: "Automatic disconnect of sessions for remote-access technologies after a specific period of inactivity",
				},
				{
					ID:          "12_3_9",
					Name:        "12.3.9",
					Description: "Activation of remote-access technologies for vendors and business partners only when needed by vendors and business partners, with immediate deactivation after use",
				},
				{
					ID:          "12_3_10",
					Name:        "12.3.10",
					Description: "For personnel accessing cardholder data via remote-access technologies, prohibit the copying, moving, and storage of cardholder data onto local hard drives and removable electronic media, unless explicitly authorized for a defined business need. Where there is an authorized business need, the usage policies must require the data be protected in accordance with all applicable PCI DSS Requirements.",
				},
				{
					ID:          "12_4",
					Name:        "12.4",
					Description: "Ensure that the security policy and procedures clearly define information security responsibilities for all personnel.",
				},
				{
					ID:          "12_4_1",
					Name:        "12.4.1",
					Description: "Additional requirement for service providers only: Executive management shall establish responsibility for the protection of cardholder data and a PCI DSS compliance program to include: • Overall accountability for maintaining PCI DSS compliance • Defining a charter for a PCI DSS compliance program and communication to executive management  Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
				{
					ID:          "12_5",
					Name:        "12.5",
					Description: "Assign to an individual or team the following information security management responsibilities:",
				},
				{
					ID:          "12_5_1",
					Name:        "12.5.1",
					Description: "Establish, document, and distribute security policies and procedures.",
				},
				{
					ID:          "12_5_2",
					Name:        "12.5.2",
					Description: "Monitor and analyze security alerts and information, and distribute to appropriate personnel.",
				},
				{
					ID:          "12_5_3",
					Name:        "12.5.3",
					Description: "Establish, document, and distribute security incident response and escalation procedures to ensure timely and effective handling of all situations.",
				},
				{
					ID:          "12_5_4",
					Name:        "12.5.4",
					Description: "Administer user accounts, including additions, deletions, and modifications.",
				},
				{
					ID:          "12_5_5",
					Name:        "12.5.5",
					Description: "Monitor and control all access to data.",
				},
				{
					ID:          "12_6",
					Name:        "12.6",
					Description: "Implement a formal security awareness program to make all personnel aware of the cardholder data security policy and procedures.",
				},
				{
					ID:          "12_6_1",
					Name:        "12.6.1",
					Description: "Educate personnel upon hire and at least annually. Note: Methods can vary depending on the role of the personnel and their level of access to the cardholder data.",
				},
				{
					ID:          "12_6_2",
					Name:        "12.6.2",
					Description: "Require personnel to acknowledge at least annually that they have read and understood the security policy and procedures.",
				},
				{
					ID:          "12_7",
					Name:        "12.7",
					Description: "Screen potential personnel prior to hire to minimize the risk of attacks from internal sources. (Examples of background checks include previous employment history, criminal record, credit history, and reference checks.)  Note: For those potential personnel to be hired for certain positions such as store cashiers who only have access to one card number at a time when facilitating a transaction, this requirement is a recommendation only.",
				},
				{
					ID:          "12_8",
					Name:        "12.8",
					Description: "Maintain and implement policies and procedures to manage service providers, with whom cardholder data is shared, or that could affect the security of cardholder data, as follows",
				},
				{
					ID:          "12_8_1",
					Name:        "12.8.1",
					Description: "Maintain a list of service providers including a description of the service provided.",
				},
				{
					ID:          "12_8_2",
					Name:        "12.8.2",
					Description: "Maintain a written agreement that includes an acknowledgement that the service providers are responsible for the security of cardholder data the service providers possess or otherwise store, process or transmit on behalf of the customer, or to the extent that they could impact the security of the customer’s cardholder data environment.  Note: The exact wording of an acknowledgement will depend on the agreement between the two parties, the details of the service being provided, and the responsibilities assigned to each party. The acknowledgement does not have to include the exact wording provided in this requirement.",
				},
				{
					ID:          "12_8_3",
					Name:        "12.8.3",
					Description: "Ensure there is an established process for engaging service providers including proper due diligence prior to engagement.",
				},
				{
					ID:          "12_8_4",
					Name:        "12.8.4",
					Description: "Maintain a program to monitor service providers’ PCI DSS compliance status at least annually.",
				},
				{
					ID:          "12_8_5",
					Name:        "12.8.5",
					Description: "Maintain information about which PCI DSS requirements are managed by each service provider, and which are managed by the entity.",
				},
				{
					ID:          "12_9",
					Name:        "12.9",
					Description: "Additional requirement for service providers only: Service providers acknowledge in writing to customers that they are responsible for the security of cardholder data the service provider possesses or otherwise stores, processes, or transmits on behalf of the customer, or to the extent that they could impact the security of the customer’s cardholder data environment.  Note: The exact wording of an acknowledgement will depend on the agreement between the two parties, the details of the service being provided, and the responsibilities assigned to each party. The acknowledgement does not have to include the exact wording provided in this requirement.",
				},
				{
					ID:          "12_10",
					Name:        "12.10",
					Description: "Implement an incident response plan. Be prepared to respond immediately to a system breach.",
				},
				{
					ID:          "12_10_1",
					Name:        "12.10.1",
					Description: "Create the incident response plan to be implemented in the event of system breach. Ensure the plan addresses the following, at a minimum: • Roles, responsibilities, and communication and contact strategies in the event of a compromise including notification of the payment brands, at a minimum • Specific incident response procedures • Business recovery and continuity procedures • Data backup processes • Analysis of legal requirements for reporting compromises • Coverage and responses of all critical system components • Reference or inclusion of incident response procedures from the payment brands.",
				},
				{
					ID:          "12_10_2",
					Name:        "12.10.2",
					Description: "Review and test the plan, including all elements listed in Requirement 12.10.1, at least annually.",
				},
				{
					ID:          "12_10_3",
					Name:        "12.10.3",
					Description: "Designate specific personnel to be available on a 24/7 basis to respond to alerts.",
				},
				{
					ID:          "12_10_4",
					Name:        "12.10.4",
					Description: "Provide appropriate training to staff with security breach response responsibilities.",
				},
				{
					ID:          "12_10_5",
					Name:        "12.10.5",
					Description: "Include alerts from security monitoring systems, including but not limited to intrusion-detection, intrusion-prevention, firewalls, and file-integrity monitoring systems.",
				},
				{
					ID:          "12_10_6",
					Name:        "12.10.6",
					Description: "Develop a process to modify and evolve the incident response plan according to lessons learned and to incorporate industry developments.",
				},
				{
					ID:          "12_11",
					Name:        "12.11",
					Description: "Additional requirement for service providers only: Perform reviews at least quarterly to confirm personnel are following security policies and operational procedures. Reviews must cover the following processes: • Daily log reviews • Firewall rule-set reviews • Applying configuration standards to new systems • Responding to security alerts • Change management processes Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
				{
					ID:          "12_11_1",
					Name:        "12.11.1",
					Description: "Additional requirement for service providers only:  Maintain documentation of quarterly review process to include: • Documenting results of the reviews • Review and sign off of results by personnel assigned responsibility for the PCI DSS compliance program Note: This requirement is a best practice until January 31, 2018, after which it becomes a requirement.",
				},
			},
		},
		{
			ID:          "A1",
			Name:        "A1",
			Description: "Additional PCI DSS Requirements for Shared Hosting Providers",
			Controls: []Control{
				{
					ID:          "A1",
					Name:        "A1",
					Description: "Protect each entity’s (that is, merchant, service provider, or other entity) hosted environment and data, per A1.1 through A1.4: A hosting provider must fulfill these requirements as well as all other relevant sections of the PCI DSS.  Note: Even though a hosting provider may meet these requirements, the compliance of the entity that uses the hosting provider is not guaranteed. Each entity must comply with the PCI DSS and validate compliance as applicable.",
				},
				{
					ID:          "A1_1",
					Name:        "A1.1",
					Description: "Ensure that each entity only runs processes that have access to that entity’s cardholder data environment.",
				},
				{
					ID:          "A1_2",
					Name:        "A1.2",
					Description: "Restrict each entity’s access and privileges to its own cardholder data environment only.",
				},
				{
					ID:          "A1_3",
					Name:        "A1.3",
					Description: "Ensure logging and audit trails are enabled and unique to each entity’s cardholder data environment and consistent with PCI DSS Requirement 10.",
				},
				{
					ID:          "A1_4",
					Name:        "A1.4",
					Description: "Enable processes to provide for timely forensic investigation in the event of a compromise to any hosted merchant or service provider.",
				},
			},
		},
		{
			ID:          "A2",
			Name:        "A2",
			Description: "Additional PCI DSS Requirements for Entities using SSL/Early TLS for Card-Present POS POI Terminal Connections",
			Controls: []Control{
				{
					ID:          "A2_1",
					Name:        "A2.1",
					Description: "Where POS POI terminals (and the SSL/TLS termination points to which they connect) use SSL and/or early TLS, the entity must either • Confirm the devices are not susceptible to any known exploits for those protocols. Or: • Have a formal Risk Mitigation and Migration Plan in place.",
				},
				{
					ID:          "A2_2",
					Name:        "A2.2",
					Description: "Entities with existing implementations (other than as allowed in A2.1) that use SSL and/or early TLS must have a formal Risk Mitigation and Migration Plan in place.",
				},
				{
					ID:          "A2_3",
					Name:        "A2.3",
					Description: "Additional Requirement for Service Providers Only: All service providers must provide a secure service offering by June 30, 2016.  Note: Prior to June 30, 2016, the service provider must either have a secure protocol option included in their service offering, or have a documented Risk Mitigation and Migration Plan (per A2.2) that includes a target date for provision of a secure protocol option no later than June 30, 2016. After this date, all service providers must offer a secure protocol option for their service.",
				},
			},
		},
	},
}

func init() {
	utils.Must(RegisterStandard(&pciDss3_2))
}
