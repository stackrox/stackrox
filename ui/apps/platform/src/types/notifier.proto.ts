import { KeyValuePair } from './common.proto';
import { PolicySeverity } from './policy.proto';
import { Traits } from './traits.proto';

export type NotifierIntegration =
    | AWSSecurityHubNotifierIntegration
    | CSCCNotifierIntegration
    | EmailNotifierIntegration
    | GenericNotifierIntegration
    | JiraNotifierIntegration
    | PagerDutyNotifierIntegration
    | SplunkNotifierIntegration
    | SumoLogicNotifierIntegration
    | SyslogNotifierIntegration;

export type BaseNotifierIntegration = {
    id: string;
    name: string;
    type: string;
    uiEndpoint: string;
    labelKey: string;
    labelDefault: string;
    traits?: Traits;
};

/*
 * The server masks the value of scrub:always stored credentials in responses and logs.
 */

// awsSecurityHub

export type AWSSecurityHubNotifierIntegration = {
    type: 'awsSecurityHub';
    awsSecurityHub: AWSSecurityHub;
} & BaseNotifierIntegration;

export type AWSSecurityHub = {
    region: string;
    credentials: AWSSecurityHubCredentials;
    accountId: string;
};

export type AWSSecurityHubCredentials = {
    accessKeyId: string; // scrub:always
    secretAccessKey: string; // scrub:always
    stsEnabled: boolean;
};

// cscc

export type CSCCNotifierIntegration = {
    type: 'cscc';
    cscc: CSCC;
} & BaseNotifierIntegration;

export type CSCC = {
    serviceAccount: string; // scrub:always
    sourceId: string;
    wifEnabled: boolean;
};

// email

export type EmailNotifierIntegration = {
    type: 'email';
    email: Email;
} & BaseNotifierIntegration;

export type Email = {
    server: string; // scrub:dependent
    sender: string;
    username: string; // scrub:dependent
    password: string; // scrub: always
    disableTLS: boolean;
    // DEPRECATED_useStartTLS deprecated
    from: string;
    startTLSAuthMethod: EmailAuthMethod;
    allowUnauthenticatedSmtp: boolean;
};

export type EmailAuthMethod = 'DISABLED' | 'PLAIN' | 'LOGIN';

// generic

export type GenericNotifierIntegration = {
    type: 'generic';
    generic: Generic;
} & BaseNotifierIntegration;

export type Generic = {
    endpoint: string; // scrub:dependent validate:nolocalendpoint
    skipTLSVerify: boolean;
    caCert: string;
    username: string; // scrub:dependent
    password: string; // scrub:always
    headers: KeyValuePair[];
    extraFields: KeyValuePair[];
    auditLoggingEnabled: boolean;
};

// jira

export type JiraNotifierIntegration = {
    type: 'jira';
    jira: Jira;
} & BaseNotifierIntegration;

export type Jira = {
    url: string; // scrub:dependent validate:nolocalendpoint
    username: string; // scrub:dependent
    password: string; // scrub:always
    issueType: string;
    priorityMappings: JiraPriorityMapping[];
    defaultFieldsJson: string;
};

export type JiraPriorityMapping = {
    severity: PolicySeverity;
    priorityName: string;
};

// pagerduty

export type PagerDutyNotifierIntegration = {
    type: 'pagerduty';
    pagerduty: PagerDuty;
} & BaseNotifierIntegration;

export type PagerDuty = {
    apiKey: string; // scrub:always
};

// splunk

export type SplunkNotifierIntegration = {
    type: 'splunk';
    splunk: Splunk;
} & BaseNotifierIntegration;

export type Splunk = {
    httpToken: string; // scrub:always
    httpEndpoint: string; // scrub:always validate:nolocalendpoint
    insecure: boolean;
    truncate: string; // int64
    auditLoggingEnabled: boolean;
    // derivedSourceType deprecated
    sourceTypes: Record<string, string>;
};

// sumologic

export type SumoLogicNotifierIntegration = {
    type: 'sumologic';
    sumologic: SumoLogic;
} & BaseNotifierIntegration;

export type SumoLogic = {
    httpSourceAddress: string; // validate:nolocalendpoint
    skipTLSVerify: boolean;
};

// syslog

export type SyslogNotifierIntegration = {
    type: 'syslog';
    syslog: Syslog;
} & BaseNotifierIntegration;

// Eventually this will support TCP, UDP, and local endpoints
export type Syslog = SyslogTCP;

export type SyslogCEFOptions = 'CEF' | 'LEGACY' | null;

export type SyslogBase = {
    messageFormat?: SyslogCEFOptions;
    localFacility?: SyslogLocalFacility;
    extraFields: KeyValuePair[];
};

export type SyslogLocalFacility =
    | 'LOCAL0'
    | 'LOCAL1'
    | 'LOCAL2'
    | 'LOCAL3'
    | 'LOCAL4'
    | 'LOCAL5'
    | 'LOCAL6'
    | 'LOCAL7';

export type SyslogTCP = {
    tcpConfig: SyslogTCPConfig;
} & SyslogBase;

export type SyslogTCPConfig = {
    hostname: string; // scrub:dependent
    port: number; // int32
    skipTlsVerify: boolean;
    useTls: boolean;
};
