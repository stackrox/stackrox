# Notifiers

A notifier is used in StackRox to send notifications to a third-party system such as Splunk, Microsoft Sentinel
or PagerDuty.

## Overview

Each notifier must implement the interfaces of the notifications it wants to support. Following notifier types exist.

### Types of notifiers

| Type                    | Interface                                  | Description                                                                                                                                   |
|-------------------------|--------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| AlertNotifier           | pkg/notifiers/alert_notifier.go            | The alert notifications used to send alerts generated by StackRox's policy engine. Examples are the Microsoft Sentinel or PagerDuty notifier. |
| AuditNotifier           | pkg/notifiers/audit_notifier.go            | The AuditLog notifications are used to send notifications about AuditLogs.                                                                    |
| NetworkPolicyNotifier   | pkg/notifiers/network_policy_notifier.go   | NetworkPolicyNotifier sends notifications about Network Policies.                                                                             |
| ResolvableAlertNotifier | pkg/notifiers/resolvable_alert_notifier.go | The ResolvableAlertNotifier is used to resolve alerts from a third party system. PagerDuty and AWS Security Hub implement these.              |
| ReportNotifier          | pkg/notifiers/report_notifier.go           | The Report notifier defines to send reports, e.g. acscs email and email support this type.                                                    |


## Write a notifier

To write a notifier you have to follow these steps:

1. Create a new pkg for your notifier in `central/notifiers`, e.g. `externalsystem`.
2. Create a new Go file, `externalsystem/my_notifier.go` and add a struct which implements one of the interfaces above, e.g. the `AlertNotifier` interface.
3. Import the new package in `central/notifiers/all/all.go`.
4. Depending on your needs of custom data, create a new configuration in the [Notifier](https://github.com/stackrox/stackrox/blob/master/proto/storage/notifier.proto#L20-L31) message for your notifier.
5. Register the notifier in the `init` func of the `externalsystem/my_notifier.go` Go file by using the `notifiers.Add` function. You can find several examples of this in other notifier implementations.
6. Implement the functions of the interface, create the client for the external service and try to send a message.
7. Implement the `Test` function with example data to trigger an alert with a simple HTTP call. This can be called via sending the config to `/v1/notifiers/test -X POST --data $notifier -H "Content-Type: application/json".
8. Implement encryption, see next `Encryption` chapter.
9. Use the admin events logger, see `Admin Events Logger` chapter.

### Admin Events Logger

To display logs of a notifier in the Admin Events overview in StackRox you need to use the logger as [here](https://github.com/stackrox/stackrox/blob/master/central/notifiers/acscsemail/acscsemail.go#L26).

```
log = logging.LoggerForModule(option.EnableAdministrationEvents())
```

This will display the logs in StackRox under `/main/administration-events?s[Resource%20Type]=Notifier` in the UI.

### Encryption

Encryption of notifier secrets is used in ACS Cloud Service. The encryption feature is disabled by default and is enabled by setting `ROX_ENC_NOTIFIER_CREDS=true`.
An example PR can be found [here](https://github.com/stackrox/stackrox/pull/12829).

For this you need to:

1. Add your notifier to `central/notifiers/utils/encryption.go` to return the credentials of the notifier.
2. Add test cases to `central/notifiers/utils/encryption_test.go`.
3. Load the encryption keys in the notifier's `init` function and protect it by the env setting `env.EncNotifierCreds.BooleanSetting()`, `ROX_ENC_NOTIFIER_CREDS`.
4. Test it by enabling notifier secrets in StackRox by running `./../../dev-tools/setup-notifier-encryption.sh`.
5. Create a new notifier with a secret and check that data doesn't contain unencrypted data by running: `go run tools/deserialize-proto/main.go --type storage.Notifier --id <UUID>`