import table from '../selectors/table';
import toast from '../selectors/toast';
import tooltip from '../selectors/tooltip';
import navigationSelectors from '../selectors/navigation';

export const url = '/main/integrations';

export const selectors = {
    configure: `${navigationSelectors.navExpandable}:contains("Platform Configuration")`,
    navLink: `${navigationSelectors.nestedNavLinks}:contains("Integrations")`,
    breadcrumbItem: '.pf-c-breadcrumb__item',
    title1: 'h1', // for example, append :contains("Integrations")
    title2: 'h2', // for example, append :contains("${integrationLabel}")
    tile: 'a[data-testid="integration-tile"]',
    tableRowNameLink: 'tbody td a', // TODO td[data-label="Name"] would be even better, but no dataLabel prop yet
    clusters: {
        k8sCluster0: 'div.rt-td:contains("Kubernetes Cluster 0")',
    },
    buttons: {
        newApiToken: 'a:contains("Generate token")',
        newClusterInitBundle: 'a:contains("Generate bundle")',
        next: 'button:contains("Next")',
        downloadYAML: 'button:contains("Download YAML")',
        delete: 'button:contains("Delete")',
        test: 'button:contains("Test")',
        create: 'button:contains("Create")',
        save: 'button:contains("Save")',
        confirm: 'button:contains("Confirm")',
        generate: 'button:contains("Generate")',
        back: 'button:contains("Back")',
        revoke: 'button:contains("Revoke")',
        closePanel: 'button[data-testid="cancel"]',
        newIntegration: 'a:contains("New integration")',
    },
    apiTokenForm: {
        nameInput: 'form[data-testid="api-token-form"] input[name="name"]',
        roleSelect: 'form[data-testid="api-token-form"] .react-select__control',
    },
    apiTokenBox: 'span:contains("eyJ")', // all API tokens start with eyJ
    apiTokenDetailsDiv: 'div[data-testid="api-token-details"]',
    clusterForm: {
        nameInput: 'form[data-testid="cluster-form"] input[name="name"]',
        imageInput: 'form[data-testid="cluster-form"] input[name="mainImage"]',
        endpointInput: 'form[data-testid="cluster-form"] input[name="centralApiEndpoint"]',
    },
    dockerRegistryForm: {
        nameInput: "form input[name='name']",
        typesSelect: 'form .react-select__control',
        endpointInput: "form input[name='docker.endpoint']",
    },
    slackForm: {
        nameInput: "form input[name='name']",
        defaultWebhook: "form input[name='labelDefault']",
        labelAnnotationKey: "form input[name='labelKey']",
    },
    awsSecurityHubForm: {
        nameInput: "form input[name='name']",
        awsAccountNumber: "form input[name='awsSecurityHub.accountId']",
        awsRegion: 'form .react-select__control',
        awsRegionListItems: '.react-select__menu-list > div',
        awsAccessKeyId: "form input[name='awsSecurityHub.credentials.accessKeyId']",
        awsSecretAccessKey: "form input[name='awsSecurityHub.credentials.secretAccessKey']",
    },
    syslogForm: {
        nameInput: "form input[name='name']",
        localFacility: 'form .react-select__control',
        localFacilityListItems: '.react-select__menu-list > div',
        receiverHost: "form input[name='syslog.tcpConfig.hostname']",
        receiverPort: 'form .react-numeric-input input',
        useTls: "form input[name='syslog.tcpConfig.useTls']",
        disableTlsValidation: "form input[name='syslog.tcpConfig.skipTlsVerify']",
    },
    modalHeader: '.ReactModal__Content header',
    formSaveButton: 'button[data-testid="save-integration"]',
    resultsSection: '[data-testid="results-message"]',
    labeledValue: '[data-testid="labeled-value"]',
    plugins: '#image-integrations a[data-testid="integration-tile"]',
    dialog: '.dialog',
    checkboxes: 'input',
    table,
    toast,
    tooltip,
};

export const labels = {
    authProviders: {
        apitoken: 'API Token',
        clusterInitBundle: 'Cluster Init Bundle',
    },
    backups: {
        gcs: 'Google Cloud Storage',
        s3: 'Amazon S3',
    },
    imageIntegrations: {
        artifactory: 'JFrog Artifactory',
        artifactregistry: 'Google Artifact Registry',
        azure: 'Microsoft ACR',
        clair: 'CoreOS Clair',
        clairify: 'StackRox Scanner',
        docker: 'Generic Docker Registry',
        ecr: 'Amazon ECR',
        google: 'Google Container Registry',
        ibm: 'IBM Cloud',
        nexus: 'Sonatype Nexus',
        quay: 'Quay.io',
        rhel: 'Red Hat',
    },
    notifiers: {
        awsSecurityHub: 'AWS Security Hub',
        cscc: 'Google Cloud SCC',
        email: 'Email',
        generic: 'Generic Webhook',
        jira: 'Jira',
        pagerduty: 'PagerDuty',
        slack: 'Slack',
        splunk: 'Splunk',
        sumologic: 'Sumo Logic',
        syslog: 'Syslog',
        teams: 'Microsoft Teams',
    },
    signatureIntegrations: {
        signature: 'Signature',
    },
};
