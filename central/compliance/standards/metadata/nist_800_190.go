package metadata

var nist800_190 = Standard{
	ID:   "NIST_800_190",
	Name: "NIST 800-190",
	Categories: []Category{
		{
			ID:          "4_1",
			Name:        "4.1",
			Description: "Image Countermeasures",
			Controls: []Control{
				{
					ID:          "4_1",
					Name:        "4.1",
					Description: "Image Countermeasures",
				},
				{
					ID:          "4_1_1",
					Name:        "4.1.1",
					Description: "Image Vulnerabilities\nOrganizations should use tools that take the pipeline-based build approach and immutable nature\nof containers and images into their design to provide more actionable and reliable results. Key\naspects of effective tools and processes include:\n1. Integration with the entire lifecycle of images, from the beginning of the build process, to\nwhatever registries the organization is using, to runtime.\n2. Visibility into vulnerabilities at all layers of the image, not just the base layer of the\nimage but also application frameworks and custom software the organization is using.\nVisibility should be centralized across the organization and provide flexible reporting and\nmonitoring views aligned with organizations’ business processes.\n3. Policy-driven enforcement; organizations should be able to create “quality gates” at each\nstage of the build and deployment process to ensure that only images that meet the\norganization’s vulnerability and configuration policies are allowed to progress. For\nexample, organizations should be able to configure a rule in the build process to prevent\nthe progression of images that include vulnerabilities with Common Vulnerability\nScoring System (CVSS) [18] ratings above a selected threshold.",
				},
				{
					ID:          "4_1_2",
					Name:        "4.1.2",
					Description: "Image configuration defects\nOrganizations should adopt tools and processes to validate and enforce compliance with secure configuration best practices. For example, images should be configured to run as non-privileged users. Tools and processes that should be adopted include:\n1. Validation of image configuration settings, including vendor recommendations and third- party best practices.\n2. Ongoing, continuously updated, centralized reporting and monitoring of image compliance state to identify weaknesses and risks at the organizational level.\n3. Enforcement of compliance requirements by optionally preventing the running of non- compliant images.\n4. Use of base layers from trusted sources only, frequent updates of base layers, and selection of base layers from minimalistic technologies like Alpine Linux and Windows Nano Server to reduce attack surface areas.A final recommendation for image configuration is that SSH and other remote administration tools designed to provide remote shells to hosts should never be enabled within containers. Containers should be run in an immutable manner to derive the greatest security benefit from their use. Enabling remote access to them via these tools implies a degree of change that violates this principle and exposes them to greater risk of network-based attack. Instead, all remote management of containers should be done through the container runtime APIs, which may be accessed via orchestration tools, or by creating remote shell sessions to the host on which the container is running.",
				},
				{
					ID:          "4_1_3",
					Name:        "4.1.3",
					Description: "Embedded malware. Organizations should continuously monitor all images for embedded malware. The monitoring processes should include the use of malware signature sets and behavioral detection heuristics based largely on actual “in the wild” attacks.",
				},
				{
					ID:          "4_1_4",
					Name:        "4.1.4",
					Description: "Embedded clear text secrets\nSecrets should be stored outside of images and provided dynamically at runtime as needed. Most orchestrators, such as Docker Swarm and Kubernetes, include native management of secrets. These orchestrators not only provide secure storage of secrets and ‘just in time’ injection to containers, but also make it much simpler to integrate secret management into the build and deployment processes. For example, an organization could use these tools to securely provision the database connection string into a web application container. The orchestrator can ensure that only the web application container had access to this secret, that it is not persisted to disk, and that anytime the web app is deployed, the secret is provisioned into it.\nOrganizations may also integrate their container deployments with existing enterprise secret management systems that are already in use for storing secrets in non-container environments. These tools typically provide APIs to retrieve secrets securely as containers are deployed, which eliminates the need to persist them within images.\nRegardless of the tool chosen, organizations should ensure that secrets are only provided to the specific containers that require them, based on a pre-defined and administrator-controlled setting, and that secrets are always encrypted at rest and in transit using Federal Information Processing Standard (FIPS) 140 approved cryptographic algorithms5 contained in validated cryptographic modules.",
				},
				{
					ID:          "4_1_5",
					Name:        "4.1.5",
					Description: "Use of untrusted images\nOrganizations should maintain a set of trusted images and registries and ensure that only images from this set are allowed to run in their environment, thus mitigating the risk of untrusted or malicious components being deployed.\nTo mitigate these risks, organizations should take a multilayered approach that includes:\nCapability to centrally control exactly what images and registries are trusted in their environment;\nDiscrete identification of each image by cryptographic signature, using a NIST-validated implementation6;\nEnforcement to ensure that all hosts in the environment only run images from these approved lists;\nValidation of image signatures before image execution to ensure images are from trusted sources and have not been tampered with; and\nOngoing monitoring and maintenance of these repositories to ensure images within them are maintained and updated as vulnerabilities and configuration requirements change.",
				},
			},
		},
		{
			ID:          "4_2",
			Name:        "4.2",
			Description: "Registry Countermeasures",
			Controls: []Control{
				{
					ID:          "4_2",
					Name:        "4.2",
					Description: "Registry Countermeasures",
				},
				{
					ID:          "4_2_1",
					Name:        "4.2.1",
					Description: "Insecure connections to registries\n Organizations should configure their development tools, orchestrators, and container runtimes to\nonly connect to registries over encrypted channels. The specific steps vary between tools, but the key goal is to ensure that all data pushed to and pulled from a registry occurs between trusted endpoints and is encrypted in transit.",
				},
				{
					ID:          "4_2_2",
					Name:        "4.2.2",
					Description: "Stale images in registries\nThe risk of using stale images can be mitigated through two primary methods. First, organizations can prune registries of unsafe, vulnerable images that should no longer be used. This process can be automated based on time triggers and labels associated with images. Second, operational practices should emphasize accessing images using immutable names that specify discrete versions of images to be used. For example, rather than configuring a deployment job to use the image called my-app, configure it to deploy specific versions of the image, such as my-app:2.3 and my-app:2.4 to ensure that specific, known good instances of images are deployed as part of each job.\nAnother option is using a “latest” tag for images and referencing this tag in deployment automation. However, because this tag is only a label attached to the image and not a guarantee of freshness, organizations should be cautious to not overly trust it. Regardless of whether an organization chooses to use discrete names or to use a “latest” tag, it is critical that processes be put in place to ensure that either the automation is using the most recent unique name or the images tagged “latest” actually do represent the most up-to-date versions.",
				},
				{
					ID:          "4_2_3",
					Name:        "4.2.3",
					Description: "Insufficient authentication and authorization restrictions\nAll access to registries that contain proprietary or sensitive images should require authentication. Any write access to a registry should require authentication to ensure that only images from trusted entities can be added to it. For example, only allow developers to push images to the specific repositories they are responsible for, rather than being able to update any repository.Organizations should consider federating with existing accounts, such as their own or a cloud provider’s directory service to take advantage of security controls already in place for those accounts. All write access to registries should be audited and any read actions for sensitive images should similarly be logged.\nRegistries also provide an opportunity to apply context-aware authorization controls to actions. For example, organizations can configure their continuous integration processes to allow images to be signed by the authorized personnel and pushed to a registry only after they have passed a vulnerability scan and compliance assessment. Organizations should integrate these automated scans into their processes to prevent the promotion and deployment of vulnerable or misconfigured images.",
				},
			},
		},
		{
			ID:          "4_3",
			Name:        "4.3",
			Description: "Orchestrator Countermeasures",
			Controls: []Control{
				{
					ID:          "4_3",
					Name:        "4.3",
					Description: "Orchestrator Countermeasures",
				},
				{
					ID:          "4_3_1",
					Name:        "4.3.1",
					Description: "Unbounded administrative access\nEspecially because of their wide-ranging span of control, orchestrators should use a least privilege access model in which users are only granted the ability to perform the specific actions on the specific hosts, containers, and images their job roles require. For example, members of the test team should only be given access to the images used in testing and the hosts used for running them, and should only be able to manipulate the containers they created. Test team members should have limited or no access to containers used in production",
				},
				{
					ID:          "4_3_2_",
					Name:        "4.3.2.",
					Description: "Unauthorized access\nAccess to cluster-wide administrative accounts should be tightly controlled as these accounts provide ability to affect all resources in the environment. Organizations should use strong authentication methods, such as requiring multifactor authentication instead of just a password.\nOrganizations should implement single sign-on to existing directory systems where applicable. Single sign-on simplifies the orchestrator authentication experience, makes it easier for users to use strong authentication credentials, and centralizes auditing of access, making anomaly detection more effective.\nTraditional approaches for data at rest encryption often involve the use of host-based capabilities that may be incompatible with containers. Thus, organizations should use tools for encrypting data used with containers that allow the data to be accessed properly from containers regardless of the node they are running on. Such encryption tools should provide the same barriers to unauthorized access and tampering, using the same cryptographic approaches as those defined in NIST SP 800-111",
				},
				{
					ID:          "4_3_3",
					Name:        "4.3.3",
					Description: "Poorly separated inter-container network traffic\nOrchestrators should be configured to separate network traffic into discrete virtual networks by sensitivity level. While per-app segmentation is also possible, for most organizations and use cases, simply defining networks by sensitivity level provides sufficient mitigation of risk with a manageable degree of complexity. For example, public-facing apps can share a virtual network,internal apps can use another, and communication between the two should occur through a small number of well-defined interfaces.",
				},
				{
					ID:          "4_3_4",
					Name:        "4.3.4",
					Description: "Mixing of workload sensitivity levels Orchestrators should be configured to isolate deployments to specific sets of hosts by sensitivity levels. The particular approach for implementing this varies depending on the orchestrator in use, but the general model is to define rules that prevent high sensitivity workloads from being placed on the same host as those running lower sensitivity workloads. This can be accomplished through the use of host ‘pinning’ within the orchestrator or even simply by having separate, individually managed clusters for each sensitivity level.",
				},
				{
					ID:          "4_3_5",
					Name:        "4.3.5",
					Description: "Orchestrator node trust. Orchestration platforms should be configured to provide features that create a secure environment for all the apps they run. Orchestrators should ensure that nodes are securely introduced to the cluster, have a persistent identity throughout their lifecycle, and can also provide an accurate inventory of nodes and their connectivity states. Organizations should ensure that orchestration platforms are designed specifically to be resilient to compromise of individual nodes without compromising the overall security of the cluster. A compromised node must be able to be isolated and removed from the cluster without disrupting or degrading overall cluster operations. Finally, organizations should choose orchestrators that provide mutually authenticated network connections between cluster members and end-to-end encryption of intra- cluster traffic. Because of the portability of containers, many deployments may occur across networks organizations do not directly control, so a secure-by-default posture is particularly important for this scenario.",
				},
			},
		},
		{
			ID:          "4_4",
			Name:        "4.4",
			Description: "Container Countermeasures",
			Controls: []Control{
				{
					ID:          "4_4",
					Name:        "4.4",
					Description: "Container Countermeasures",
				},
				{
					ID:          "4_4_1",
					Name:        "4.4.1",
					Description: "Vulnerabilities within the runtime software\nThe container runtime must be carefully monitored for vulnerabilities, and when problems are detected, they must be remediated quickly. A vulnerable runtime exposes all containers it supports, as well as the host itself, to potentially significant risk. Organizations should use tools to look for Common Vulnerabilities and Exposures (CVEs) vulnerabilities in the runtimes deployed, to upgrade any instances at risk, and to ensure that orchestrators only allow deployments to properly maintained runtimes.",
				},
				{
					ID:          "4_4_2",
					Name:        "4.4.2",
					Description: "Unbounded network access from containers\nOrganizations should control the egress network traffic sent by containers. At minimum, these controls should be in place at network borders, ensuring containers are not able to send traffic across networks of differing sensitivity levels, such as from an environment hosting secure data to the internet, similar to the patterns used for traditional architectures. This dynamic rule management is critical due to the scale and rate of change of containerized apps, as well as their ephemeral networking topology.\nSpecifically, app-aware tools should provide the following capabilities:\n• Automated determination of proper container networking surfaces, including both inbound ports and process-port bindings;\n• Detection of traffic flows both between containers and other network entities, over both ‘on the wire’ traffic and encapsulated traffic; and\n• Detection of network anomalies, such as unexpected traffic flows within the organization’s network, port scanning, or outbound access to potentially dangerous destinations.",
				},
				{
					ID:          "4_4_3",
					Name:        "4.4.3",
					Description: "Insecure container runtime configurations\nOrganizations should automate compliance with container runtime configuration  Documented technical implementation guidance, such as the Center for Internet Security Docker Benchmark [20], provides details on options and recommended settings, but operationalizing this guidance depends on automation. Organizations can use a variety of tools to “scan” and assess their compliance at a point in time, but such approaches do not scale. Instead, organizations should use tools or processes that continuously assess configuration settings across the environment and actively enforce them.",
				},
				{
					ID:          "4_4_4",
					Name:        "4.4.4",
					Description: "App vulnerabilities\nExisting host-based intrusion detection processes and tools are often unable to detect and prevent attacks within containers due to the differing technical architecture and operational practices. These profiles should then be able to prevent and detect anomalies at runtime, including events such as:\n• Invalid or unexpected process execution,\n• Invalid or unexpected system calls,\n• Changes to protected configuration files and binaries,\n• Writes to unexpected locations and file types,\n• Creation of unexpected network listeners,\n• Traffic sent to unexpected network destinations, and\n• Malware storage or execution.\nContainers should also be run with their root filesystems in read-only mode.",
				},
				{
					ID:          "4_4_5",
					Name:        "4.4.5",
					Description: "Rogue containers\nOrganizations should institute separate environments for development, test, production, and other scenarios, each with specific controls to provide role-based access control for container deployment and management activities. All container creation should be associated with individual user identities and logged to provide a clear audit trail of activity. Further, organizations are encouraged to use security tools that can enforce baseline requirements for vulnerability management and compliance prior to allowing an image to be run",
				},
			},
		},
		{
			ID:          "4_5",
			Name:        "4.5",
			Description: "Host OS Countermeasures",
			Controls: []Control{
				{
					ID:          "4_5",
					Name:        "4.5",
					Description: "Host OS Countermeasures",
				},
				{
					ID:          "4_5_1",
					Name:        "4.5.1",
					Description: "Large attack surface\nFor organizations using container-specific OSs, the threats are typically more minimal to start with since the OSs are specifically designed to host containers and have other services and functionality disabled. Further, because these optimized OSs are designed specifically for hosting containers, they typically feature read-only file systems and employ other hardening practices by default. Whenever possible, organizations should use these minimalistic OSs to reduce their attack surfaces and mitigate the typical risks and hardening activities associated with general-purpose OSs.\nOrganizations that cannot use a container-specific OS should follow the guidance in NIST SP 800-123, Guide to General Server Security [23] to reduce the attack surface of their hosts as much as possible. For example, hosts that run containers should only run containers and not run other apps, like a web server or database, outside of containers. The host OS should not run unnecessary system services, such as a print spooler, that increase its attack and patching surface areas. Finally, hosts should be continuously scanned for vulnerabilities and updates applied quickly, not just to the container runtime but also to lower-level components such as the kernel that containers rely upon for secure, compartmentalized operation.",
				},
				{
					ID:          "4_5_2",
					Name:        "4.5.2",
					Description: "Shared kernel\nIn addition to grouping container workloads onto hosts by sensitivity level, organizations should not mix containerized and non-containerized workloads on the same host instance. For example, if a host is running a web server container, it should not also run a web server (or any other app) as a regularly installed component directly within the host OS. Keeping containerized workloads isolated to container-specific hosts makes it simpler and safer to apply countermeasures and defenses that are optimized for protecting containers.",
				},
				{
					ID:          "4_5_3",
					Name:        "4.5.3",
					Description: "Host OS component vulnerabilities\nOrganizations should implement management practices and tools to validate the versioning of components provided for base OS management and functionality. Even though container- specific OSs have a much more minimal set of components than general-purpose OSs, they still do have vulnerabilities and still require remediation. Organizations should use tools provided by the OS vendor or other trusted organizations to regularly check for and apply updates to all software components used within the OS. The OS should be kept up to date not only with security updates, but also the latest component updates recommended by the vendor. This is particularly important for the kernel and container runtime components as newer releases of these components often add additional security protections and capabilities beyond simply correcting vulnerabilities. Some organizations may choose to simply redeploy new OS instances with the necessary updates, rather than updating existing systems. This approach is also valid, although it often requires more sophisticated operational practices.\nHost OSs should be operated in an immutable manner with no data or state stored uniquely and persistently on the host and no application-level dependencies provided by the host. Instead, all app components and dependencies should be packaged and deployed in containers. This enables the host to be operated in a nearly stateless manner with a greatly reduced attack surface. Additionally, it provides a more trustworthy way to identify anomalies and configuration drift.",
				},
				{
					ID:          "4_5_4",
					Name:        "4.5.4",
					Description: "Improper user access rights\nThough most container deployments rely on orchestrators to distribute jobs across hosts, organizations should still ensure that all authentication to the OS is audited, login anomalies are monitored, and any escalation to perform privileged operations is logged. This makes it possible to identify anomalous access patterns such as an individual logging on to a host directly and running privileged commands to manipulate containers.",
				},
				{
					ID:          "4_5_5",
					Name:        "4.5.5",
					Description: "Host file system tampering\nEnsure that containers are run with the minimal set of file system permissions required. Very rarely should containers mount local file systems on a host. Instead, any file changes that containers need to persist to disk should be made within storage volumes specifically allocated for this purpose. In no case should containers be able to mount sensitive directories on a host’s file system, especially those containing configuration settings for the operating system.",
				},
			},
		},
		{
			ID:          "4_6",
			Name:        "4.6",
			Description: "Hardware Countermeasures",
			Controls: []Control{
				{
					ID:          "4_6",
					Name:        "4.6",
					Description: "Hardware countermeasures\nTo NIST, “trusted” means that the platform behaves as it is expected to: the software inventory is accurate, the configuration settings and security controls are in place and operating as they should, and so on. “Trusted” also means that it is known that no unauthorized person has tampered with the software or its configuration on the hosts. Hardware root of trust is not a concept unique to containers, but container management and security tools can leverage attestations for the rest of the container technology architecture to ensure containers are being run in secure environments.\nThe currently available way to provide trusted computing is to:\n1. Measure firmware, software, and configuration data before it is executed using a Root of Trust for Measurement (RTM).\n2. Store those measurements in a hardware root of trust, like a trusted platform module (TPM).\n3. Validate that the current measurements match the expected measurements. If so, it can be attested that the platform can be trusted to behave as expected.\nTPM-enabled devices can check the integrity of the machine during the boot process, enabling protection and detection mechanisms to function in hardware, at pre-boot, and in the secure boot process. This same trust and integrity assurance can be extended beyond the OS and the boot loader to the container runtimes and apps. Note that while standards are being developed to enable verification of hardware trust by users of cloud services, not all clouds expose this functionality to their customers. In cases where technical verification is not provided, organizations should address hardware trust requirements as part of their service agreements with cloud providers.\nFor container technologies, these techniques are currently applicable at the hardware, hypervisor, and host OS layers, with early work in progress to apply these to container-specific components.",
				},
			},
		},
	},
}

func init() {
	AllStandards = append(AllStandards, nist800_190)
}
