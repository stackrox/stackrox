-- Write your migrate up statements here

CREATE EXTENSION IF NOT EXISTS pg_stat_statements WITH SCHEMA public;

CREATE TABLE active_components (
    id character varying NOT NULL,
    deploymentid uuid,
    componentid character varying,
    serialized bytea
);

CREATE TABLE active_components_active_contexts_slices (
    active_components_id character varying NOT NULL,
    idx integer NOT NULL,
    containername character varying,
    imageid character varying
);

CREATE TABLE administration_events (
    id uuid NOT NULL,
    type integer,
    level integer,
    domain character varying,
    resource_type character varying,
    numoccurrences bigint,
    lastoccurredat timestamp without time zone,
    createdat timestamp without time zone,
    serialized bytea
);

CREATE TABLE alerts (
    id uuid NOT NULL,
    policy_id character varying,
    policy_name character varying,
    policy_description character varying,
    policy_disabled boolean,
    policy_categories text[],
    policy_severity integer,
    policy_enforcementactions integer[],
    policy_lastupdated timestamp without time zone,
    policy_sortname character varying,
    policy_sortlifecyclestage character varying,
    policy_sortenforcement boolean,
    lifecyclestage integer,
    clusterid uuid,
    clustername character varying,
    namespace character varying,
    namespaceid uuid,
    deployment_id uuid,
    deployment_name character varying,
    deployment_inactive boolean,
    image_id character varying,
    image_name_registry character varying,
    image_name_remote character varying,
    image_name_tag character varying,
    image_name_fullname character varying,
    resource_resourcetype integer,
    resource_name character varying,
    enforcement_action integer,
    "time" timestamp without time zone,
    state integer,
    platformcomponent boolean,
    entitytype integer,
    serialized bytea
);

CREATE TABLE api_tokens (
    id character varying NOT NULL,
    expiration timestamp without time zone,
    revoked boolean,
    serialized bytea
);

CREATE TABLE auth_machine_to_machine_configs (
    id uuid NOT NULL,
    issuer character varying,
    serialized bytea
);

CREATE TABLE auth_machine_to_machine_configs_mappings (
    auth_machine_to_machine_configs_id uuid NOT NULL,
    idx integer NOT NULL,
    role character varying
);

CREATE TABLE auth_providers (
    id character varying NOT NULL,
    name character varying,
    serialized bytea
);

CREATE TABLE blobs (
    name character varying NOT NULL,
    length bigint,
    modifiedtime timestamp without time zone,
    serialized bytea
);

CREATE TABLE cloud_sources (
    id uuid NOT NULL,
    name character varying,
    type integer,
    serialized bytea
);

CREATE TABLE cluster_cve_edges (
    id character varying NOT NULL,
    isfixable boolean,
    fixedby character varying,
    clusterid uuid,
    cveid character varying,
    serialized bytea
);

CREATE TABLE cluster_cves (
    id character varying NOT NULL,
    cvebaseinfo_cve character varying,
    cvebaseinfo_publishedon timestamp without time zone,
    cvebaseinfo_createdat timestamp without time zone,
    cvss numeric,
    severity integer,
    impactscore numeric,
    snoozed boolean,
    snoozeexpiry timestamp without time zone,
    type integer,
    serialized bytea
);

CREATE TABLE cluster_health_statuses (
    id uuid NOT NULL,
    sensorhealthstatus integer,
    collectorhealthstatus integer,
    overallhealthstatus integer,
    admissioncontrolhealthstatus integer,
    scannerhealthstatus integer,
    lastcontact timestamp without time zone,
    serialized bytea
);

CREATE TABLE cluster_init_bundles (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE clusters (
    id uuid NOT NULL,
    name character varying,
    type integer,
    labels jsonb,
    status_providermetadata_cluster_type integer,
    status_orchestratormetadata_version character varying,
    serialized bytea
);

CREATE TABLE collections (
    id character varying NOT NULL,
    name character varying,
    createdby_name character varying,
    updatedby_name character varying,
    serialized bytea
);

CREATE TABLE collections_embedded_collections (
    collections_id character varying NOT NULL,
    idx integer NOT NULL,
    id character varying
);

CREATE TABLE compliance_configs (
    standardid character varying NOT NULL,
    serialized bytea
);

CREATE TABLE compliance_domains (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE compliance_integrations (
    id uuid NOT NULL,
    version character varying,
    clusterid uuid,
    operatorinstalled boolean,
    operatorstatus integer,
    serialized bytea
);

CREATE TABLE compliance_operator_benchmark_v2 (
    id uuid NOT NULL,
    name character varying,
    version character varying,
    shortname character varying,
    serialized bytea
);

CREATE TABLE compliance_operator_benchmark_v2_profiles (
    compliance_operator_benchmark_v2_id uuid NOT NULL,
    idx integer NOT NULL,
    profilename character varying,
    profileversion character varying
);

CREATE TABLE compliance_operator_check_result_v2 (
    id character varying NOT NULL,
    checkid character varying,
    checkname character varying,
    clusterid uuid,
    status integer,
    severity integer,
    createdtime timestamp without time zone,
    scanconfigname character varying,
    rationale character varying,
    scanrefid uuid,
    rulerefid uuid,
    serialized bytea
);

CREATE TABLE compliance_operator_check_results (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE compliance_operator_cluster_scan_config_statuses (
    id uuid NOT NULL,
    clusterid uuid,
    scanconfigid uuid,
    lastupdatedtime timestamp without time zone,
    serialized bytea
);

CREATE TABLE compliance_operator_profile_v2 (
    id character varying NOT NULL,
    profileid character varying,
    name character varying,
    profileversion character varying,
    producttype character varying,
    standard character varying,
    clusterid uuid,
    profilerefid uuid,
    serialized bytea
);

CREATE TABLE compliance_operator_profile_v2_rules (
    compliance_operator_profile_v2_id character varying NOT NULL,
    idx integer NOT NULL,
    rulename character varying
);

CREATE TABLE compliance_operator_profiles (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE compliance_operator_remediation_v2 (
    id uuid NOT NULL,
    name character varying,
    compliancecheckresultname character varying,
    clusterid character varying,
    serialized bytea
);

CREATE TABLE compliance_operator_report_snapshot_v2 (
    reportid uuid NOT NULL,
    scanconfigurationid uuid,
    name character varying,
    reportstatus_runstate integer,
    reportstatus_startedat timestamp without time zone,
    reportstatus_completedat timestamp without time zone,
    reportstatus_reportrequesttype integer,
    reportstatus_reportnotificationmethod integer,
    user_id character varying,
    user_name character varying,
    serialized bytea
);

CREATE TABLE compliance_operator_report_snapshot_v2_scans (
    compliance_operator_report_snapshot_v2_reportid uuid NOT NULL,
    idx integer NOT NULL,
    scanrefid character varying,
    laststartedtime timestamp without time zone
);

CREATE TABLE compliance_operator_rule_v2 (
    id character varying NOT NULL,
    name character varying,
    ruletype character varying,
    severity integer,
    clusterid uuid,
    rulerefid uuid,
    serialized bytea
);

CREATE TABLE compliance_operator_rule_v2_controls (
    compliance_operator_rule_v2_id character varying NOT NULL,
    idx integer NOT NULL,
    standard character varying,
    control character varying
);

CREATE TABLE compliance_operator_rules (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE compliance_operator_scan_configuration_v2 (
    id uuid NOT NULL,
    scanconfigname character varying,
    modifiedby_name character varying,
    serialized bytea
);

CREATE TABLE compliance_operator_scan_configuration_v2_clusters (
    compliance_operator_scan_configuration_v2_id uuid NOT NULL,
    idx integer NOT NULL,
    clusterid uuid
);

CREATE TABLE compliance_operator_scan_configuration_v2_notifiers (
    compliance_operator_scan_configuration_v2_id uuid NOT NULL,
    idx integer NOT NULL,
    id character varying
);

CREATE TABLE compliance_operator_scan_configuration_v2_profiles (
    compliance_operator_scan_configuration_v2_id uuid NOT NULL,
    idx integer NOT NULL,
    profilename character varying
);

CREATE TABLE compliance_operator_scan_setting_binding_v2 (
    id character varying NOT NULL,
    name character varying,
    clusterid uuid,
    scansettingname character varying,
    serialized bytea
);

CREATE TABLE compliance_operator_scan_setting_bindings (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE compliance_operator_scan_v2 (
    id character varying NOT NULL,
    scanconfigname character varying,
    clusterid uuid,
    profile_profilerefid uuid,
    status_result character varying,
    lastexecutedtime timestamp without time zone,
    scanname character varying,
    scanrefid uuid,
    laststartedtime timestamp without time zone,
    serialized bytea
);

CREATE TABLE compliance_operator_scans (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE compliance_operator_suite_v2 (
    id uuid NOT NULL,
    name character varying,
    clusterid uuid,
    serialized bytea
);

CREATE TABLE compliance_run_metadata (
    runid character varying NOT NULL,
    standardid character varying,
    clusterid uuid,
    finishtimestamp timestamp without time zone,
    serialized bytea
);

CREATE TABLE compliance_run_results (
    runmetadata_runid character varying NOT NULL,
    runmetadata_standardid character varying,
    runmetadata_clusterid uuid,
    runmetadata_finishtimestamp timestamp without time zone,
    serialized bytea
);

CREATE TABLE compliance_strings (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE configs (
    serialized bytea
);

CREATE TABLE declarative_config_healths (
    id uuid NOT NULL,
    serialized bytea
);

CREATE TABLE delegated_registry_configs (
    serialized bytea
);

CREATE TABLE deployments (
    id uuid NOT NULL,
    name character varying,
    type character varying,
    namespace character varying,
    namespaceid uuid,
    orchestratorcomponent boolean,
    labels jsonb,
    podlabels jsonb,
    created timestamp without time zone,
    clusterid uuid,
    clustername character varying,
    annotations jsonb,
    priority bigint,
    imagepullsecrets text[],
    serviceaccount character varying,
    serviceaccountpermissionlevel integer,
    riskscore numeric,
    platformcomponent boolean,
    serialized bytea
);

CREATE TABLE deployments_containers (
    deployments_id uuid NOT NULL,
    idx integer NOT NULL,
    image_id character varying,
    image_name_registry character varying,
    image_name_remote character varying,
    image_name_tag character varying,
    image_name_fullname character varying,
    securitycontext_privileged boolean,
    securitycontext_dropcapabilities text[],
    securitycontext_addcapabilities text[],
    securitycontext_readonlyrootfilesystem boolean,
    resources_cpucoresrequest numeric,
    resources_cpucoreslimit numeric,
    resources_memorymbrequest numeric,
    resources_memorymblimit numeric
);

CREATE TABLE deployments_containers_envs (
    deployments_id uuid NOT NULL,
    deployments_containers_idx integer NOT NULL,
    idx integer NOT NULL,
    key character varying,
    value character varying,
    envvarsource integer
);

CREATE TABLE deployments_containers_secrets (
    deployments_id uuid NOT NULL,
    deployments_containers_idx integer NOT NULL,
    idx integer NOT NULL,
    name character varying,
    path character varying
);

CREATE TABLE deployments_containers_volumes (
    deployments_id uuid NOT NULL,
    deployments_containers_idx integer NOT NULL,
    idx integer NOT NULL,
    name character varying,
    source character varying,
    destination character varying,
    readonly boolean,
    type character varying
);

CREATE TABLE deployments_ports (
    deployments_id uuid NOT NULL,
    idx integer NOT NULL,
    containerport integer,
    protocol character varying,
    exposure integer
);

CREATE TABLE deployments_ports_exposure_infos (
    deployments_id uuid NOT NULL,
    deployments_ports_idx integer NOT NULL,
    idx integer NOT NULL,
    level integer,
    servicename character varying,
    serviceport integer,
    nodeport integer,
    externalips text[],
    externalhostnames text[]
);

CREATE TABLE discovered_clusters (
    id uuid NOT NULL,
    metadata_name character varying,
    metadata_type integer,
    metadata_firstdiscoveredat timestamp without time zone,
    status integer,
    sourceid uuid,
    lastupdatedat timestamp without time zone,
    serialized bytea
);

CREATE TABLE external_backups (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE groups (
    props_id character varying NOT NULL,
    props_authproviderid character varying,
    props_key character varying,
    props_value character varying,
    rolename character varying,
    serialized bytea
);

CREATE TABLE hashes (
    clusterid character varying NOT NULL,
    serialized bytea
);

CREATE TABLE image_component_cve_edges (
    id character varying NOT NULL,
    isfixable boolean,
    fixedby character varying,
    imagecomponentid character varying,
    imagecveid character varying,
    serialized bytea
);

CREATE TABLE image_component_edges (
    id character varying NOT NULL,
    location character varying,
    imageid character varying,
    imagecomponentid character varying,
    serialized bytea
);

CREATE TABLE image_components (
    id character varying NOT NULL,
    name character varying,
    version character varying,
    priority bigint,
    source integer,
    riskscore numeric,
    topcvss numeric,
    operatingsystem character varying,
    serialized bytea
);

CREATE TABLE image_cve_edges (
    id character varying NOT NULL,
    firstimageoccurrence timestamp without time zone,
    state integer,
    imageid character varying,
    imagecveid character varying,
    serialized bytea
);

CREATE TABLE image_cves (
    id character varying NOT NULL,
    cvebaseinfo_cve character varying,
    cvebaseinfo_publishedon timestamp without time zone,
    cvebaseinfo_createdat timestamp without time zone,
    operatingsystem character varying,
    cvss numeric,
    severity integer,
    impactscore numeric,
    snoozed boolean,
    snoozeexpiry timestamp without time zone,
    nvdcvss numeric,
    serialized bytea
);

CREATE TABLE image_integrations (
    id uuid NOT NULL,
    name character varying,
    clusterid uuid,
    serialized bytea
);

CREATE TABLE images (
    id character varying NOT NULL,
    name_registry character varying,
    name_remote character varying,
    name_tag character varying,
    name_fullname character varying,
    metadata_v1_created timestamp without time zone,
    metadata_v1_user character varying,
    metadata_v1_command text[],
    metadata_v1_entrypoint text[],
    metadata_v1_volumes text[],
    metadata_v1_labels jsonb,
    scan_scantime timestamp without time zone,
    scan_operatingsystem character varying,
    signature_fetched timestamp without time zone,
    components integer,
    cves integer,
    fixablecves integer,
    lastupdated timestamp without time zone,
    priority bigint,
    riskscore numeric,
    topcvss numeric,
    serialized bytea
);

CREATE TABLE images_layers (
    images_id character varying NOT NULL,
    idx integer NOT NULL,
    instruction character varying,
    value character varying
);

CREATE TABLE installation_infos (
    serialized bytea
);

CREATE TABLE integration_healths (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE k8s_roles (
    id uuid NOT NULL,
    name character varying,
    namespace character varying,
    clusterid uuid,
    clustername character varying,
    clusterrole boolean,
    labels jsonb,
    annotations jsonb,
    serialized bytea
);

CREATE TABLE listening_endpoints (
    id uuid NOT NULL,
    port bigint,
    protocol integer,
    closetimestamp timestamp without time zone,
    processindicatorid uuid,
    closed boolean,
    deploymentid uuid,
    poduid uuid,
    clusterid uuid,
    namespace character varying,
    serialized bytea
);

CREATE TABLE log_imbues (
    id character varying NOT NULL,
    "timestamp" timestamp without time zone,
    serialized bytea
);

CREATE TABLE namespaces (
    id uuid NOT NULL,
    name character varying,
    clusterid uuid,
    clustername character varying,
    labels jsonb,
    annotations jsonb,
    serialized bytea
);

CREATE TABLE network_baselines (
    deploymentid uuid NOT NULL,
    clusterid uuid,
    namespace character varying,
    serialized bytea
);

CREATE TABLE network_entities (
    info_id character varying NOT NULL,
    info_externalsource_cidr cidr,
    info_externalsource_default boolean,
    info_externalsource_discovered boolean,
    serialized bytea
);

CREATE TABLE network_flows_v2 (
    flow_id bigint NOT NULL,
    props_srcentity_type integer,
    props_srcentity_id character varying,
    props_dstentity_type integer,
    props_dstentity_id character varying,
    props_dstport integer,
    props_l4protocol integer,
    lastseentimestamp timestamp without time zone,
    clusterid character varying NOT NULL
)
PARTITION BY LIST (clusterid);

CREATE SEQUENCE network_flows_v2_flow_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE network_flows_v2_flow_id_seq OWNED BY network_flows_v2.flow_id;

CREATE TABLE network_graph_configs (
    id character varying NOT NULL,
    serialized bytea
);

CREATE TABLE networkpolicies (
    id character varying NOT NULL,
    clusterid uuid,
    namespace character varying,
    serialized bytea
);

CREATE TABLE networkpoliciesundodeployments (
    deploymentid uuid NOT NULL,
    serialized bytea
);

CREATE TABLE networkpolicyapplicationundorecords (
    clusterid uuid NOT NULL,
    serialized bytea
);

CREATE TABLE node_component_edges (
    id character varying NOT NULL,
    nodeid uuid,
    nodecomponentid character varying,
    serialized bytea
);

CREATE TABLE node_components (
    id character varying NOT NULL,
    name character varying,
    version character varying,
    priority bigint,
    riskscore numeric,
    topcvss numeric,
    operatingsystem character varying,
    serialized bytea
);

CREATE TABLE node_components_cves_edges (
    id character varying NOT NULL,
    isfixable boolean,
    fixedby character varying,
    nodecomponentid character varying,
    nodecveid character varying,
    serialized bytea
);

CREATE TABLE node_cves (
    id character varying NOT NULL,
    cvebaseinfo_cve character varying,
    cvebaseinfo_publishedon timestamp without time zone,
    cvebaseinfo_createdat timestamp without time zone,
    operatingsystem character varying,
    cvss numeric,
    severity integer,
    impactscore numeric,
    snoozed boolean,
    snoozeexpiry timestamp without time zone,
    orphaned boolean,
    orphanedtime timestamp without time zone,
    serialized bytea
);

CREATE TABLE nodes (
    id uuid NOT NULL,
    name character varying,
    clusterid uuid,
    clustername character varying,
    labels jsonb,
    annotations jsonb,
    joinedat timestamp without time zone,
    containerruntime_version character varying,
    osimage character varying,
    lastupdated timestamp without time zone,
    scan_scantime timestamp without time zone,
    components integer,
    cves integer,
    fixablecves integer,
    priority bigint,
    riskscore numeric,
    topcvss numeric,
    serialized bytea
);

CREATE TABLE nodes_taints (
    nodes_id uuid NOT NULL,
    idx integer NOT NULL,
    key character varying,
    value character varying,
    tainteffect integer
);

CREATE TABLE notification_schedules (
    serialized bytea
);

CREATE TABLE notifier_enc_configs (
    serialized bytea
);

CREATE TABLE notifiers (
    id character varying NOT NULL,
    name character varying,
    serialized bytea
);

CREATE TABLE permission_sets (
    id uuid NOT NULL,
    name character varying,
    serialized bytea
);

CREATE TABLE pods (
    id uuid NOT NULL,
    name character varying,
    deploymentid uuid,
    namespace character varying,
    clusterid uuid,
    serialized bytea
);

CREATE TABLE pods_live_instances (
    pods_id uuid NOT NULL,
    idx integer NOT NULL,
    imagedigest character varying
);

CREATE TABLE policies (
    id character varying NOT NULL,
    name character varying,
    description character varying,
    disabled boolean,
    categories text[],
    lifecyclestages integer[],
    severity integer,
    enforcementactions integer[],
    lastupdated timestamp without time zone,
    sortname character varying,
    sortlifecyclestage character varying,
    sortenforcement boolean,
    serialized bytea
);

CREATE TABLE policy_categories (
    id character varying NOT NULL,
    name character varying,
    serialized bytea
);

CREATE TABLE policy_category_edges (
    id character varying NOT NULL,
    policyid character varying,
    categoryid character varying,
    serialized bytea
);

CREATE TABLE process_baseline_results (
    deploymentid uuid NOT NULL,
    clusterid uuid,
    namespace character varying,
    serialized bytea
);

CREATE TABLE process_baselines (
    id character varying NOT NULL,
    key_deploymentid uuid,
    key_clusterid uuid,
    key_namespace character varying,
    serialized bytea
);

CREATE TABLE process_indicators (
    id uuid NOT NULL,
    deploymentid uuid,
    containername character varying,
    podid character varying,
    poduid uuid,
    signal_containerid character varying,
    signal_time timestamp without time zone,
    signal_name character varying,
    signal_args character varying,
    signal_execfilepath character varying,
    signal_uid bigint,
    clusterid uuid,
    namespace character varying,
    serialized bytea
);

CREATE TABLE report_configurations (
    id character varying NOT NULL,
    name character varying,
    type integer,
    scopeid character varying,
    resourcescope_collectionid character varying,
    creator_name character varying,
    serialized bytea
);

CREATE TABLE report_configurations_notifiers (
    report_configurations_id character varying NOT NULL,
    idx integer NOT NULL,
    id character varying
);

CREATE TABLE report_snapshots (
    reportid uuid NOT NULL,
    reportconfigurationid character varying,
    name character varying,
    reportstatus_runstate integer,
    reportstatus_queuedat timestamp without time zone,
    reportstatus_completedat timestamp without time zone,
    reportstatus_reportrequesttype integer,
    reportstatus_reportnotificationmethod integer,
    requester_id character varying,
    requester_name character varying,
    serialized bytea
);

CREATE TABLE risks (
    id character varying NOT NULL,
    subject_namespace character varying,
    subject_clusterid uuid,
    subject_type integer,
    score numeric,
    serialized bytea
);

CREATE TABLE role_bindings (
    id uuid NOT NULL,
    name character varying,
    namespace character varying,
    clusterid uuid,
    clustername character varying,
    clusterrole boolean,
    labels jsonb,
    annotations jsonb,
    roleid uuid,
    serialized bytea
);

CREATE TABLE role_bindings_subjects (
    role_bindings_id uuid NOT NULL,
    idx integer NOT NULL,
    kind integer,
    name character varying
);

CREATE TABLE roles (
    name character varying NOT NULL,
    serialized bytea
);

CREATE TABLE secrets (
    id uuid NOT NULL,
    name character varying,
    clusterid uuid,
    clustername character varying,
    namespace character varying,
    createdat timestamp without time zone,
    serialized bytea
);

CREATE TABLE secrets_files (
    secrets_id uuid NOT NULL,
    idx integer NOT NULL,
    type integer,
    cert_enddate timestamp without time zone
);

CREATE TABLE secrets_files_registries (
    secrets_id uuid NOT NULL,
    secrets_files_idx integer NOT NULL,
    idx integer NOT NULL,
    name character varying
);

CREATE TABLE secured_units (
    id uuid NOT NULL,
    "timestamp" timestamp without time zone,
    numnodes bigint,
    numcpuunits bigint,
    serialized bytea
);

CREATE TABLE sensor_upgrade_configs (
    serialized bytea
);

CREATE TABLE service_accounts (
    id uuid NOT NULL,
    name character varying,
    namespace character varying,
    clustername character varying,
    clusterid uuid,
    labels jsonb,
    annotations jsonb,
    serialized bytea
);

CREATE TABLE service_identities (
    serialstr character varying NOT NULL,
    serialized bytea
);

CREATE TABLE signature_integrations (
    id character varying NOT NULL,
    name character varying,
    serialized bytea
);

CREATE TABLE simple_access_scopes (
    id uuid NOT NULL,
    name character varying,
    serialized bytea
);

CREATE TABLE system_infos (
    backupinfo_requestor_name character varying,
    serialized bytea
);

CREATE TABLE versions (
    seqnum integer,
    version character varying,
    lastpersisted timestamp without time zone,
    minseqnum integer,
    serialized bytea
);

CREATE TABLE vulnerability_requests (
    id character varying NOT NULL,
    name character varying,
    targetstate integer,
    status integer,
    expired boolean,
    requestor_name character varying,
    createdat timestamp without time zone,
    lastupdated timestamp without time zone,
    scope_imagescope_registry character varying,
    scope_imagescope_remote character varying,
    scope_imagescope_tag character varying,
    requesterv2_id character varying,
    requesterv2_name character varying,
    deferralreq_expiry_expireson timestamp without time zone,
    deferralreq_expiry_expireswhenfixed boolean,
    deferralreq_expiry_expirytype integer,
    cves_cves text[],
    deferralupdate_cves text[],
    falsepositiveupdate_cves text[],
    serialized bytea
);

CREATE TABLE vulnerability_requests_approvers (
    vulnerability_requests_id character varying NOT NULL,
    idx integer NOT NULL,
    name character varying
);

CREATE TABLE vulnerability_requests_approvers_v2 (
    vulnerability_requests_id character varying NOT NULL,
    idx integer NOT NULL,
    id character varying,
    name character varying
);

CREATE TABLE vulnerability_requests_comments (
    vulnerability_requests_id character varying NOT NULL,
    idx integer NOT NULL,
    user_name character varying
);

CREATE TABLE watched_images (
    name character varying NOT NULL,
    serialized bytea
);

ALTER TABLE ONLY network_flows_v2 ALTER COLUMN flow_id SET DEFAULT nextval('network_flows_v2_flow_id_seq'::regclass);

ALTER TABLE ONLY active_components_active_contexts_slices
    ADD CONSTRAINT active_components_active_contexts_slices_pkey PRIMARY KEY (active_components_id, idx);

ALTER TABLE ONLY active_components
    ADD CONSTRAINT active_components_pkey PRIMARY KEY (id);

ALTER TABLE ONLY administration_events
    ADD CONSTRAINT administration_events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY alerts
    ADD CONSTRAINT alerts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY api_tokens
    ADD CONSTRAINT api_tokens_pkey PRIMARY KEY (id);

ALTER TABLE ONLY auth_machine_to_machine_configs_mappings
    ADD CONSTRAINT auth_machine_to_machine_configs_mappings_pkey PRIMARY KEY (auth_machine_to_machine_configs_id, idx);

ALTER TABLE ONLY auth_machine_to_machine_configs
    ADD CONSTRAINT auth_machine_to_machine_configs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY auth_providers
    ADD CONSTRAINT auth_providers_pkey PRIMARY KEY (id);

ALTER TABLE ONLY blobs
    ADD CONSTRAINT blobs_pkey PRIMARY KEY (name);

ALTER TABLE ONLY cloud_sources
    ADD CONSTRAINT cloud_sources_pkey PRIMARY KEY (id);

ALTER TABLE ONLY cluster_cve_edges
    ADD CONSTRAINT cluster_cve_edges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY cluster_cves
    ADD CONSTRAINT cluster_cves_pkey PRIMARY KEY (id);

ALTER TABLE ONLY cluster_health_statuses
    ADD CONSTRAINT cluster_health_statuses_pkey PRIMARY KEY (id);

ALTER TABLE ONLY cluster_init_bundles
    ADD CONSTRAINT cluster_init_bundles_pkey PRIMARY KEY (id);

ALTER TABLE ONLY clusters
    ADD CONSTRAINT clusters_pkey PRIMARY KEY (id);

ALTER TABLE ONLY collections_embedded_collections
    ADD CONSTRAINT collections_embedded_collections_pkey PRIMARY KEY (collections_id, idx);

ALTER TABLE ONLY collections
    ADD CONSTRAINT collections_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_configs
    ADD CONSTRAINT compliance_configs_pkey PRIMARY KEY (standardid);

ALTER TABLE ONLY compliance_domains
    ADD CONSTRAINT compliance_domains_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_integrations
    ADD CONSTRAINT compliance_integrations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_benchmark_v2
    ADD CONSTRAINT compliance_operator_benchmark_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_benchmark_v2_profiles
    ADD CONSTRAINT compliance_operator_benchmark_v2_profiles_pkey PRIMARY KEY (compliance_operator_benchmark_v2_id, idx);

ALTER TABLE ONLY compliance_operator_check_result_v2
    ADD CONSTRAINT compliance_operator_check_result_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_check_results
    ADD CONSTRAINT compliance_operator_check_results_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_cluster_scan_config_statuses
    ADD CONSTRAINT compliance_operator_cluster_scan_config_statuses_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_profile_v2
    ADD CONSTRAINT compliance_operator_profile_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_profile_v2_rules
    ADD CONSTRAINT compliance_operator_profile_v2_rules_pkey PRIMARY KEY (compliance_operator_profile_v2_id, idx);

ALTER TABLE ONLY compliance_operator_profiles
    ADD CONSTRAINT compliance_operator_profiles_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_remediation_v2
    ADD CONSTRAINT compliance_operator_remediation_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_report_snapshot_v2
    ADD CONSTRAINT compliance_operator_report_snapshot_v2_pkey PRIMARY KEY (reportid);

ALTER TABLE ONLY compliance_operator_report_snapshot_v2_scans
    ADD CONSTRAINT compliance_operator_report_snapshot_v2_scans_pkey PRIMARY KEY (compliance_operator_report_snapshot_v2_reportid, idx);

ALTER TABLE ONLY compliance_operator_rule_v2_controls
    ADD CONSTRAINT compliance_operator_rule_v2_controls_pkey PRIMARY KEY (compliance_operator_rule_v2_id, idx);

ALTER TABLE ONLY compliance_operator_rule_v2
    ADD CONSTRAINT compliance_operator_rule_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_rules
    ADD CONSTRAINT compliance_operator_rules_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_scan_configuration_v2_clusters
    ADD CONSTRAINT compliance_operator_scan_configuration_v2_clusters_pkey PRIMARY KEY (compliance_operator_scan_configuration_v2_id, idx);

ALTER TABLE ONLY compliance_operator_scan_configuration_v2_notifiers
    ADD CONSTRAINT compliance_operator_scan_configuration_v2_notifiers_pkey PRIMARY KEY (compliance_operator_scan_configuration_v2_id, idx);

ALTER TABLE ONLY compliance_operator_scan_configuration_v2
    ADD CONSTRAINT compliance_operator_scan_configuration_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_scan_configuration_v2_profiles
    ADD CONSTRAINT compliance_operator_scan_configuration_v2_profiles_pkey PRIMARY KEY (compliance_operator_scan_configuration_v2_id, idx);

ALTER TABLE ONLY compliance_operator_scan_setting_binding_v2
    ADD CONSTRAINT compliance_operator_scan_setting_binding_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_scan_setting_bindings
    ADD CONSTRAINT compliance_operator_scan_setting_bindings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_scan_v2
    ADD CONSTRAINT compliance_operator_scan_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_scans
    ADD CONSTRAINT compliance_operator_scans_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_operator_suite_v2
    ADD CONSTRAINT compliance_operator_suite_v2_pkey PRIMARY KEY (id);

ALTER TABLE ONLY compliance_run_metadata
    ADD CONSTRAINT compliance_run_metadata_pkey PRIMARY KEY (runid);

ALTER TABLE ONLY compliance_run_results
    ADD CONSTRAINT compliance_run_results_pkey PRIMARY KEY (runmetadata_runid);

ALTER TABLE ONLY compliance_strings
    ADD CONSTRAINT compliance_strings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY declarative_config_healths
    ADD CONSTRAINT declarative_config_healths_pkey PRIMARY KEY (id);

ALTER TABLE ONLY deployments_containers_envs
    ADD CONSTRAINT deployments_containers_envs_pkey PRIMARY KEY (deployments_id, deployments_containers_idx, idx);

ALTER TABLE ONLY deployments_containers
    ADD CONSTRAINT deployments_containers_pkey PRIMARY KEY (deployments_id, idx);

ALTER TABLE ONLY deployments_containers_secrets
    ADD CONSTRAINT deployments_containers_secrets_pkey PRIMARY KEY (deployments_id, deployments_containers_idx, idx);

ALTER TABLE ONLY deployments_containers_volumes
    ADD CONSTRAINT deployments_containers_volumes_pkey PRIMARY KEY (deployments_id, deployments_containers_idx, idx);

ALTER TABLE ONLY deployments
    ADD CONSTRAINT deployments_pkey PRIMARY KEY (id);

ALTER TABLE ONLY deployments_ports_exposure_infos
    ADD CONSTRAINT deployments_ports_exposure_infos_pkey PRIMARY KEY (deployments_id, deployments_ports_idx, idx);

ALTER TABLE ONLY deployments_ports
    ADD CONSTRAINT deployments_ports_pkey PRIMARY KEY (deployments_id, idx);

ALTER TABLE ONLY discovered_clusters
    ADD CONSTRAINT discovered_clusters_pkey PRIMARY KEY (id);

ALTER TABLE ONLY external_backups
    ADD CONSTRAINT external_backups_pkey PRIMARY KEY (id);

ALTER TABLE ONLY groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (props_id);

ALTER TABLE ONLY hashes
    ADD CONSTRAINT hashes_pkey PRIMARY KEY (clusterid);

ALTER TABLE ONLY image_component_cve_edges
    ADD CONSTRAINT image_component_cve_edges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY image_component_edges
    ADD CONSTRAINT image_component_edges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY image_components
    ADD CONSTRAINT image_components_pkey PRIMARY KEY (id);

ALTER TABLE ONLY image_cve_edges
    ADD CONSTRAINT image_cve_edges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY image_cves
    ADD CONSTRAINT image_cves_pkey PRIMARY KEY (id);

ALTER TABLE ONLY image_integrations
    ADD CONSTRAINT image_integrations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY images_layers
    ADD CONSTRAINT images_layers_pkey PRIMARY KEY (images_id, idx);

ALTER TABLE ONLY images
    ADD CONSTRAINT images_pkey PRIMARY KEY (id);

ALTER TABLE ONLY integration_healths
    ADD CONSTRAINT integration_healths_pkey PRIMARY KEY (id);

ALTER TABLE ONLY k8s_roles
    ADD CONSTRAINT k8s_roles_pkey PRIMARY KEY (id);

ALTER TABLE ONLY listening_endpoints
    ADD CONSTRAINT listening_endpoints_pkey PRIMARY KEY (id);

ALTER TABLE ONLY log_imbues
    ADD CONSTRAINT log_imbues_pkey PRIMARY KEY (id);

ALTER TABLE ONLY namespaces
    ADD CONSTRAINT namespaces_pkey PRIMARY KEY (id);

ALTER TABLE ONLY network_baselines
    ADD CONSTRAINT network_baselines_pkey PRIMARY KEY (deploymentid);

ALTER TABLE ONLY network_entities
    ADD CONSTRAINT network_entities_pkey PRIMARY KEY (info_id);

ALTER TABLE ONLY network_flows_v2
    ADD CONSTRAINT network_flows_v2_pkey PRIMARY KEY (clusterid, flow_id);

ALTER TABLE ONLY network_graph_configs
    ADD CONSTRAINT network_graph_configs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY networkpolicies
    ADD CONSTRAINT networkpolicies_pkey PRIMARY KEY (id);

ALTER TABLE ONLY networkpoliciesundodeployments
    ADD CONSTRAINT networkpoliciesundodeployments_pkey PRIMARY KEY (deploymentid);

ALTER TABLE ONLY networkpolicyapplicationundorecords
    ADD CONSTRAINT networkpolicyapplicationundorecords_pkey PRIMARY KEY (clusterid);

ALTER TABLE ONLY node_component_edges
    ADD CONSTRAINT node_component_edges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY node_components_cves_edges
    ADD CONSTRAINT node_components_cves_edges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY node_components
    ADD CONSTRAINT node_components_pkey PRIMARY KEY (id);

ALTER TABLE ONLY node_cves
    ADD CONSTRAINT node_cves_pkey PRIMARY KEY (id);

ALTER TABLE ONLY nodes
    ADD CONSTRAINT nodes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY nodes_taints
    ADD CONSTRAINT nodes_taints_pkey PRIMARY KEY (nodes_id, idx);

ALTER TABLE ONLY notifiers
    ADD CONSTRAINT notifiers_pkey PRIMARY KEY (id);

ALTER TABLE ONLY permission_sets
    ADD CONSTRAINT permission_sets_pkey PRIMARY KEY (id);

ALTER TABLE ONLY pods_live_instances
    ADD CONSTRAINT pods_live_instances_pkey PRIMARY KEY (pods_id, idx);

ALTER TABLE ONLY pods
    ADD CONSTRAINT pods_pkey PRIMARY KEY (id);

ALTER TABLE ONLY policies
    ADD CONSTRAINT policies_pkey PRIMARY KEY (id);

ALTER TABLE ONLY policy_categories
    ADD CONSTRAINT policy_categories_pkey PRIMARY KEY (id);

ALTER TABLE ONLY policy_category_edges
    ADD CONSTRAINT policy_category_edges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY process_baseline_results
    ADD CONSTRAINT process_baseline_results_pkey PRIMARY KEY (deploymentid);

ALTER TABLE ONLY process_baselines
    ADD CONSTRAINT process_baselines_pkey PRIMARY KEY (id);

ALTER TABLE ONLY process_indicators
    ADD CONSTRAINT process_indicators_pkey PRIMARY KEY (id);

ALTER TABLE ONLY report_configurations_notifiers
    ADD CONSTRAINT report_configurations_notifiers_pkey PRIMARY KEY (report_configurations_id, idx);

ALTER TABLE ONLY report_configurations
    ADD CONSTRAINT report_configurations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY report_snapshots
    ADD CONSTRAINT report_snapshots_pkey PRIMARY KEY (reportid);

ALTER TABLE ONLY risks
    ADD CONSTRAINT risks_pkey PRIMARY KEY (id);

ALTER TABLE ONLY role_bindings
    ADD CONSTRAINT role_bindings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY role_bindings_subjects
    ADD CONSTRAINT role_bindings_subjects_pkey PRIMARY KEY (role_bindings_id, idx);

ALTER TABLE ONLY roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (name);

ALTER TABLE ONLY secrets_files
    ADD CONSTRAINT secrets_files_pkey PRIMARY KEY (secrets_id, idx);

ALTER TABLE ONLY secrets_files_registries
    ADD CONSTRAINT secrets_files_registries_pkey PRIMARY KEY (secrets_id, secrets_files_idx, idx);

ALTER TABLE ONLY secrets
    ADD CONSTRAINT secrets_pkey PRIMARY KEY (id);

ALTER TABLE ONLY secured_units
    ADD CONSTRAINT secured_units_pkey PRIMARY KEY (id);

ALTER TABLE ONLY service_accounts
    ADD CONSTRAINT service_accounts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY service_identities
    ADD CONSTRAINT service_identities_pkey PRIMARY KEY (serialstr);

ALTER TABLE ONLY signature_integrations
    ADD CONSTRAINT signature_integrations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY simple_access_scopes
    ADD CONSTRAINT simple_access_scopes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY auth_machine_to_machine_configs
    ADD CONSTRAINT uni_auth_machine_to_machine_configs_issuer UNIQUE (issuer);

ALTER TABLE ONLY auth_providers
    ADD CONSTRAINT uni_auth_providers_name UNIQUE (name);

ALTER TABLE ONLY cloud_sources
    ADD CONSTRAINT uni_cloud_sources_name UNIQUE (name);

ALTER TABLE ONLY clusters
    ADD CONSTRAINT uni_clusters_name UNIQUE (name);

ALTER TABLE ONLY collections
    ADD CONSTRAINT uni_collections_name UNIQUE (name);

ALTER TABLE ONLY compliance_operator_scan_configuration_v2
    ADD CONSTRAINT uni_compliance_operator_scan_configuration_v2_scanconfigname UNIQUE (scanconfigname);

ALTER TABLE ONLY image_integrations
    ADD CONSTRAINT uni_image_integrations_name UNIQUE (name);

ALTER TABLE ONLY notifiers
    ADD CONSTRAINT uni_notifiers_name UNIQUE (name);

ALTER TABLE ONLY permission_sets
    ADD CONSTRAINT uni_permission_sets_name UNIQUE (name);

ALTER TABLE ONLY policies
    ADD CONSTRAINT uni_policies_name UNIQUE (name);

ALTER TABLE ONLY policy_categories
    ADD CONSTRAINT uni_policy_categories_name UNIQUE (name);

ALTER TABLE ONLY secured_units
    ADD CONSTRAINT uni_secured_units_timestamp UNIQUE ("timestamp");

ALTER TABLE ONLY signature_integrations
    ADD CONSTRAINT uni_signature_integrations_name UNIQUE (name);

ALTER TABLE ONLY simple_access_scopes
    ADD CONSTRAINT uni_simple_access_scopes_name UNIQUE (name);

ALTER TABLE ONLY vulnerability_requests
    ADD CONSTRAINT uni_vulnerability_requests_name UNIQUE (name);

ALTER TABLE ONLY vulnerability_requests_approvers
    ADD CONSTRAINT vulnerability_requests_approvers_pkey PRIMARY KEY (vulnerability_requests_id, idx);

ALTER TABLE ONLY vulnerability_requests_approvers_v2
    ADD CONSTRAINT vulnerability_requests_approvers_v2_pkey PRIMARY KEY (vulnerability_requests_id, idx);

ALTER TABLE ONLY vulnerability_requests_comments
    ADD CONSTRAINT vulnerability_requests_comments_pkey PRIMARY KEY (vulnerability_requests_id, idx);

ALTER TABLE ONLY vulnerability_requests
    ADD CONSTRAINT vulnerability_requests_pkey PRIMARY KEY (id);

ALTER TABLE ONLY watched_images
    ADD CONSTRAINT watched_images_pkey PRIMARY KEY (name);

CREATE INDEX activecomponents_deploymentid ON active_components USING hash (deploymentid);

CREATE INDEX activecomponentsactivecontextsslices_idx ON active_components_active_contexts_slices USING btree (idx);

CREATE INDEX alerts_deployment_id ON alerts USING hash (deployment_id);

CREATE INDEX alerts_lifecyclestage ON alerts USING btree (lifecyclestage);

CREATE INDEX alerts_policy_id ON alerts USING btree (policy_id);

CREATE INDEX alerts_sac_filter ON alerts USING btree (clusterid, namespace);

CREATE INDEX alerts_state ON alerts USING btree (state);

CREATE INDEX alerts_time ON alerts USING btree ("time");

CREATE INDEX authmachinetomachineconfigsmappings_idx ON auth_machine_to_machine_configs_mappings USING btree (idx);

CREATE INDEX clustercveedges_cveid ON cluster_cve_edges USING hash (cveid);

CREATE INDEX clustercves_cvebaseinfo_cve ON cluster_cves USING hash (cvebaseinfo_cve);

CREATE INDEX collectionsembeddedcollections_idx ON collections_embedded_collections USING btree (idx);

CREATE UNIQUE INDEX compliance_unique_indicator ON compliance_integrations USING btree (clusterid);

CREATE INDEX complianceintegrations_sac_filter ON compliance_integrations USING hash (clusterid);

CREATE INDEX complianceoperatorbenchmarkv2profiles_idx ON compliance_operator_benchmark_v2_profiles USING btree (idx);

CREATE INDEX complianceoperatorcheckresultv2_sac_filter ON compliance_operator_check_result_v2 USING hash (clusterid);

CREATE INDEX complianceoperatorclusterscanconfigstatuses_sac_filter ON compliance_operator_cluster_scan_config_statuses USING hash (clusterid);

CREATE INDEX complianceoperatorprofilev2_sac_filter ON compliance_operator_profile_v2 USING hash (clusterid);

CREATE INDEX complianceoperatorprofilev2rules_idx ON compliance_operator_profile_v2_rules USING btree (idx);

CREATE INDEX complianceoperatorremediationv2_sac_filter ON compliance_operator_remediation_v2 USING hash (clusterid);

CREATE INDEX complianceoperatorreportsnapshotv2scans_idx ON compliance_operator_report_snapshot_v2_scans USING btree (idx);

CREATE INDEX complianceoperatorrulev2_sac_filter ON compliance_operator_rule_v2 USING hash (clusterid);

CREATE INDEX complianceoperatorrulev2controls_idx ON compliance_operator_rule_v2_controls USING btree (idx);

CREATE INDEX complianceoperatorscanconfigurationv2clusters_idx ON compliance_operator_scan_configuration_v2_clusters USING btree (idx);

CREATE INDEX complianceoperatorscanconfigurationv2clusters_sac_filter ON compliance_operator_scan_configuration_v2_clusters USING hash (clusterid);

CREATE INDEX complianceoperatorscanconfigurationv2notifiers_idx ON compliance_operator_scan_configuration_v2_notifiers USING btree (idx);

CREATE INDEX complianceoperatorscanconfigurationv2profiles_idx ON compliance_operator_scan_configuration_v2_profiles USING btree (idx);

CREATE INDEX complianceoperatorscansettingbindingv2_sac_filter ON compliance_operator_scan_setting_binding_v2 USING hash (clusterid);

CREATE INDEX complianceoperatorscanv2_sac_filter ON compliance_operator_scan_v2 USING hash (clusterid);

CREATE INDEX complianceoperatorsuitev2_sac_filter ON compliance_operator_suite_v2 USING hash (clusterid);

CREATE INDEX compliancerunmetadata_sac_filter ON compliance_run_metadata USING hash (clusterid);

CREATE INDEX compliancerunresults_sac_filter ON compliance_run_results USING hash (runmetadata_clusterid);

CREATE INDEX deployments_sac_filter ON deployments USING btree (namespace, clusterid);

CREATE INDEX deploymentscontainers_idx ON deployments_containers USING btree (idx);

CREATE INDEX deploymentscontainers_image_id ON deployments_containers USING hash (image_id);

CREATE INDEX deploymentscontainersenvs_idx ON deployments_containers_envs USING btree (idx);

CREATE INDEX deploymentscontainerssecrets_idx ON deployments_containers_secrets USING btree (idx);

CREATE INDEX deploymentscontainersvolumes_idx ON deployments_containers_volumes USING btree (idx);

CREATE INDEX deploymentsports_idx ON deployments_ports USING btree (idx);

CREATE INDEX deploymentsportsexposureinfos_idx ON deployments_ports_exposure_infos USING btree (idx);

CREATE UNIQUE INDEX groups_unique_indicator ON groups USING btree (props_authproviderid, props_key, props_value, rolename);

CREATE INDEX imagecomponentcveedges_imagecomponentid ON image_component_cve_edges USING hash (imagecomponentid);

CREATE INDEX imagecomponentcveedges_imagecveid ON image_component_cve_edges USING hash (imagecveid);

CREATE INDEX imagecomponentedges_imagecomponentid ON image_component_edges USING hash (imagecomponentid);

CREATE INDEX imagecomponentedges_imageid ON image_component_edges USING hash (imageid);

CREATE INDEX imagecveedges_imagecveid ON image_cve_edges USING hash (imagecveid);

CREATE INDEX imagecveedges_imageid ON image_cve_edges USING hash (imageid);

CREATE INDEX imagecves_cvebaseinfo_cve ON image_cves USING hash (cvebaseinfo_cve);

CREATE INDEX imageintegrations_sac_filter ON image_integrations USING btree (clusterid);

CREATE INDEX imageslayers_idx ON images_layers USING btree (idx);

CREATE INDEX k8sroles_sac_filter ON k8s_roles USING btree (namespace, clusterid);

CREATE INDEX listeningendpoints_closed ON listening_endpoints USING btree (closed);

CREATE INDEX listeningendpoints_deploymentid ON listening_endpoints USING btree (deploymentid);

CREATE INDEX listeningendpoints_poduid ON listening_endpoints USING hash (poduid);

CREATE INDEX listeningendpoints_processindicatorid ON listening_endpoints USING btree (processindicatorid);

CREATE INDEX listeningendpoints_sac_filter ON listening_endpoints USING btree (clusterid, namespace);

CREATE INDEX namespaces_sac_filter ON namespaces USING btree (name, clusterid);

CREATE INDEX network_flows_dst_v2 ON ONLY network_flows_v2 USING hash (props_dstentity_id);

CREATE INDEX network_flows_lastseentimestamp_v2 ON ONLY network_flows_v2 USING brin (lastseentimestamp);

CREATE INDEX network_flows_src_v2 ON ONLY network_flows_v2 USING hash (props_srcentity_id);

CREATE INDEX networkbaselines_sac_filter ON network_baselines USING btree (clusterid, namespace);

CREATE INDEX networkentities_info_externalsource_cidr ON network_entities USING btree (info_externalsource_cidr);

CREATE INDEX networkpolicies_sac_filter ON networkpolicies USING btree (clusterid, namespace);

CREATE INDEX nodecomponentedges_nodecomponentid ON node_component_edges USING hash (nodecomponentid);

CREATE INDEX nodecomponentedges_nodeid ON node_component_edges USING hash (nodeid);

CREATE INDEX nodecomponentscvesedges_nodecomponentid ON node_components_cves_edges USING hash (nodecomponentid);

CREATE INDEX nodecomponentscvesedges_nodecveid ON node_components_cves_edges USING hash (nodecveid);

CREATE INDEX nodecves_cvebaseinfo_cve ON node_cves USING hash (cvebaseinfo_cve);

CREATE INDEX nodes_sac_filter ON nodes USING hash (clusterid);

CREATE INDEX nodestaints_idx ON nodes_taints USING btree (idx);

CREATE INDEX pods_sac_filter ON pods USING btree (namespace, clusterid);

CREATE INDEX podsliveinstances_idx ON pods_live_instances USING btree (idx);

CREATE INDEX policies_id ON policies USING btree (id);

CREATE INDEX processbaselineresults_sac_filter ON process_baseline_results USING btree (clusterid, namespace);

CREATE INDEX processbaselines_key_deploymentid ON process_baselines USING hash (key_deploymentid);

CREATE INDEX processbaselines_sac_filter ON process_baselines USING btree (key_clusterid, key_namespace);

CREATE INDEX processindicators_deploymentid ON process_indicators USING hash (deploymentid);

CREATE INDEX processindicators_poduid ON process_indicators USING hash (poduid);

CREATE INDEX processindicators_sac_filter ON process_indicators USING btree (clusterid, namespace);

CREATE INDEX processindicators_signal_time ON process_indicators USING btree (signal_time);

CREATE INDEX reportconfigurationsnotifiers_idx ON report_configurations_notifiers USING btree (idx);

CREATE INDEX risks_sac_filter ON risks USING btree (subject_namespace, subject_clusterid);

CREATE INDEX rolebindings_sac_filter ON role_bindings USING btree (namespace, clusterid);

CREATE INDEX rolebindingssubjects_idx ON role_bindings_subjects USING btree (idx);

CREATE INDEX secrets_sac_filter ON secrets USING btree (clusterid, namespace);

CREATE INDEX secretsfiles_idx ON secrets_files USING btree (idx);

CREATE INDEX secretsfilesregistries_idx ON secrets_files_registries USING btree (idx);

CREATE INDEX serviceaccounts_sac_filter ON service_accounts USING btree (namespace, clusterid);

CREATE INDEX vulnerabilityrequestsapprovers_idx ON vulnerability_requests_approvers USING btree (idx);

CREATE INDEX vulnerabilityrequestsapproversv2_idx ON vulnerability_requests_approvers_v2 USING btree (idx);

CREATE INDEX vulnerabilityrequestscomments_idx ON vulnerability_requests_comments USING btree (idx);

ALTER TABLE ONLY active_components_active_contexts_slices
    ADD CONSTRAINT fk_active_components_active_contexts_slices_active_comp0f2d0b84 FOREIGN KEY (active_components_id) REFERENCES active_components(id) ON DELETE CASCADE;

ALTER TABLE ONLY auth_machine_to_machine_configs_mappings
    ADD CONSTRAINT fk_auth_machine_to_machine_configs_mappings_auth_machin9a6c39e4 FOREIGN KEY (auth_machine_to_machine_configs_id) REFERENCES auth_machine_to_machine_configs(id) ON DELETE CASCADE;

ALTER TABLE ONLY auth_machine_to_machine_configs_mappings
    ADD CONSTRAINT fk_auth_machine_to_machine_configs_mappings_roles_ref FOREIGN KEY (role) REFERENCES roles(name) ON DELETE RESTRICT;

ALTER TABLE ONLY cluster_cve_edges
    ADD CONSTRAINT fk_cluster_cve_edges_clusters_ref FOREIGN KEY (clusterid) REFERENCES clusters(id) ON DELETE CASCADE;

ALTER TABLE ONLY collections_embedded_collections
    ADD CONSTRAINT fk_collections_embedded_collections_collections_cycle_ref FOREIGN KEY (id) REFERENCES collections(id) ON DELETE RESTRICT;

ALTER TABLE ONLY collections_embedded_collections
    ADD CONSTRAINT fk_collections_embedded_collections_collections_ref FOREIGN KEY (collections_id) REFERENCES collections(id) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_benchmark_v2_profiles
    ADD CONSTRAINT fk_compliance_operator_benchmark_v2_profiles_compliance8e6514c6 FOREIGN KEY (compliance_operator_benchmark_v2_id) REFERENCES compliance_operator_benchmark_v2(id) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_profile_v2_rules
    ADD CONSTRAINT fk_compliance_operator_profile_v2_rules_compliance_oper55f27d3c FOREIGN KEY (compliance_operator_profile_v2_id) REFERENCES compliance_operator_profile_v2(id) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_report_snapshot_v2
    ADD CONSTRAINT fk_compliance_operator_report_snapshot_v2_compliance_op4653ba9c FOREIGN KEY (scanconfigurationid) REFERENCES compliance_operator_scan_configuration_v2(id) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_report_snapshot_v2_scans
    ADD CONSTRAINT fk_compliance_operator_report_snapshot_v2_scans_complia4e9b3bd3 FOREIGN KEY (compliance_operator_report_snapshot_v2_reportid) REFERENCES compliance_operator_report_snapshot_v2(reportid) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_rule_v2_controls
    ADD CONSTRAINT fk_compliance_operator_rule_v2_controls_compliance_oper55523455 FOREIGN KEY (compliance_operator_rule_v2_id) REFERENCES compliance_operator_rule_v2(id) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_scan_configuration_v2_clusters
    ADD CONSTRAINT fk_compliance_operator_scan_configuration_v2_clusters_c45d36757 FOREIGN KEY (compliance_operator_scan_configuration_v2_id) REFERENCES compliance_operator_scan_configuration_v2(id) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_scan_configuration_v2_notifiers
    ADD CONSTRAINT fk_compliance_operator_scan_configuration_v2_notifiers_9964bf33 FOREIGN KEY (compliance_operator_scan_configuration_v2_id) REFERENCES compliance_operator_scan_configuration_v2(id) ON DELETE CASCADE;

ALTER TABLE ONLY compliance_operator_scan_configuration_v2_notifiers
    ADD CONSTRAINT fk_compliance_operator_scan_configuration_v2_notifiers_c565a2d1 FOREIGN KEY (id) REFERENCES notifiers(id) ON DELETE RESTRICT;

ALTER TABLE ONLY compliance_operator_scan_configuration_v2_profiles
    ADD CONSTRAINT fk_compliance_operator_scan_configuration_v2_profiles_c68197db9 FOREIGN KEY (compliance_operator_scan_configuration_v2_id) REFERENCES compliance_operator_scan_configuration_v2(id) ON DELETE CASCADE;

ALTER TABLE ONLY deployments_containers
    ADD CONSTRAINT fk_deployments_containers_deployments_ref FOREIGN KEY (deployments_id) REFERENCES deployments(id) ON DELETE CASCADE;

ALTER TABLE ONLY deployments_containers_envs
    ADD CONSTRAINT fk_deployments_containers_envs_deployments_containers_ref FOREIGN KEY (deployments_id, deployments_containers_idx) REFERENCES deployments_containers(deployments_id, idx) ON DELETE CASCADE;

ALTER TABLE ONLY deployments_containers_secrets
    ADD CONSTRAINT fk_deployments_containers_secrets_deployments_containers_ref FOREIGN KEY (deployments_id, deployments_containers_idx) REFERENCES deployments_containers(deployments_id, idx) ON DELETE CASCADE;

ALTER TABLE ONLY deployments_containers_volumes
    ADD CONSTRAINT fk_deployments_containers_volumes_deployments_containers_ref FOREIGN KEY (deployments_id, deployments_containers_idx) REFERENCES deployments_containers(deployments_id, idx) ON DELETE CASCADE;

ALTER TABLE ONLY deployments_ports
    ADD CONSTRAINT fk_deployments_ports_deployments_ref FOREIGN KEY (deployments_id) REFERENCES deployments(id) ON DELETE CASCADE;

ALTER TABLE ONLY deployments_ports_exposure_infos
    ADD CONSTRAINT fk_deployments_ports_exposure_infos_deployments_ports_ref FOREIGN KEY (deployments_id, deployments_ports_idx) REFERENCES deployments_ports(deployments_id, idx) ON DELETE CASCADE;

ALTER TABLE ONLY image_component_cve_edges
    ADD CONSTRAINT fk_image_component_cve_edges_image_components_ref FOREIGN KEY (imagecomponentid) REFERENCES image_components(id) ON DELETE CASCADE;

ALTER TABLE ONLY image_component_edges
    ADD CONSTRAINT fk_image_component_edges_images_ref FOREIGN KEY (imageid) REFERENCES images(id) ON DELETE CASCADE;

ALTER TABLE ONLY image_cve_edges
    ADD CONSTRAINT fk_image_cve_edges_images_ref FOREIGN KEY (imageid) REFERENCES images(id) ON DELETE CASCADE;

ALTER TABLE ONLY images_layers
    ADD CONSTRAINT fk_images_layers_images_ref FOREIGN KEY (images_id) REFERENCES images(id) ON DELETE CASCADE;

ALTER TABLE ONLY node_component_edges
    ADD CONSTRAINT fk_node_component_edges_nodes_ref FOREIGN KEY (nodeid) REFERENCES nodes(id) ON DELETE CASCADE;

ALTER TABLE ONLY node_components_cves_edges
    ADD CONSTRAINT fk_node_components_cves_edges_node_components_ref FOREIGN KEY (nodecomponentid) REFERENCES node_components(id) ON DELETE CASCADE;

ALTER TABLE ONLY nodes_taints
    ADD CONSTRAINT fk_nodes_taints_nodes_ref FOREIGN KEY (nodes_id) REFERENCES nodes(id) ON DELETE CASCADE;

ALTER TABLE ONLY pods_live_instances
    ADD CONSTRAINT fk_pods_live_instances_pods_ref FOREIGN KEY (pods_id) REFERENCES pods(id) ON DELETE CASCADE;

ALTER TABLE ONLY policy_category_edges
    ADD CONSTRAINT fk_policy_category_edges_policies_ref FOREIGN KEY (policyid) REFERENCES policies(id) ON DELETE CASCADE;

ALTER TABLE ONLY policy_category_edges
    ADD CONSTRAINT fk_policy_category_edges_policy_categories_ref FOREIGN KEY (categoryid) REFERENCES policy_categories(id) ON DELETE CASCADE;

ALTER TABLE ONLY report_configurations_notifiers
    ADD CONSTRAINT fk_report_configurations_notifiers_notifiers_ref FOREIGN KEY (id) REFERENCES notifiers(id) ON DELETE RESTRICT;

ALTER TABLE ONLY report_configurations_notifiers
    ADD CONSTRAINT fk_report_configurations_notifiers_report_configurations_ref FOREIGN KEY (report_configurations_id) REFERENCES report_configurations(id) ON DELETE CASCADE;

ALTER TABLE ONLY report_snapshots
    ADD CONSTRAINT fk_report_snapshots_report_configurations_ref FOREIGN KEY (reportconfigurationid) REFERENCES report_configurations(id) ON DELETE CASCADE;

ALTER TABLE ONLY role_bindings_subjects
    ADD CONSTRAINT fk_role_bindings_subjects_role_bindings_ref FOREIGN KEY (role_bindings_id) REFERENCES role_bindings(id) ON DELETE CASCADE;

ALTER TABLE ONLY secrets_files_registries
    ADD CONSTRAINT fk_secrets_files_registries_secrets_files_ref FOREIGN KEY (secrets_id, secrets_files_idx) REFERENCES secrets_files(secrets_id, idx) ON DELETE CASCADE;

ALTER TABLE ONLY secrets_files
    ADD CONSTRAINT fk_secrets_files_secrets_ref FOREIGN KEY (secrets_id) REFERENCES secrets(id) ON DELETE CASCADE;

ALTER TABLE ONLY vulnerability_requests_approvers_v2
    ADD CONSTRAINT fk_vulnerability_requests_approvers_v2_vulnerability_reaef85e7a FOREIGN KEY (vulnerability_requests_id) REFERENCES vulnerability_requests(id) ON DELETE CASCADE;

ALTER TABLE ONLY vulnerability_requests_approvers
    ADD CONSTRAINT fk_vulnerability_requests_approvers_vulnerability_requests_ref FOREIGN KEY (vulnerability_requests_id) REFERENCES vulnerability_requests(id) ON DELETE CASCADE;

ALTER TABLE ONLY vulnerability_requests_comments
    ADD CONSTRAINT fk_vulnerability_requests_comments_vulnerability_requests_ref FOREIGN KEY (vulnerability_requests_id) REFERENCES vulnerability_requests(id) ON DELETE CASCADE;
---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
