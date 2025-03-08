# Microsoft Sentinel Integration

In the `./data/` directory, there is a file named `example-alert.json` that contains an example of an alert sent
to Microsoft Sentinel.

| **Component**                     | **Description**                                                                                                                                                                                                  | **Link**                                                                                                          |
|-----------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| Custom Data ingestion in Sentinel | An overview of how the Log Analytics Workspace and Sentinel work together.                                                                                                                                       | [link](https://learn.microsoft.com/en-us/azure/sentinel/data-transformation)                                      |
| Logs ingestion API overview       | This API is used to ingest logs into Azure which are later consumed by Sentinel. This is the client StackRox is using to send alerts.                                                                            | [link](https://learn.microsoft.com/en-us/azure/azure-monitor/logs/logs-ingestion-api-overview)                    |
| Service Principals                | Service Principals are used to authenticate against Azure. A user needs to create one with either Secret or Certificate authentication.                                                                          | [link](https://learn.microsoft.com/en-us/entra/identity-platform/app-objects-and-service-principals?tabs=browser) |
| Data Collection Transformations   | Details about the Data Transformation. The logs hit the Data Collection Endpoint, are forwarded to Data Collection Rule which holds the transformation logic, and at last pushed to the Log Analytics Workspace. | [link](https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/data-collection-transformations)          |
| Data Collection Rule Ovierview    | Overview of Data Collection Rules.                                                                                                                                                                               | [link](https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/data-collection-rule-overview)            |

## How It Works

The integration with Microsoft Sentinel sends logs to an Azure Log Analytics workspace, which is responsible for
collecting and storing the logs. Microsoft Sentinel then consumes these logs and can generate alerts based on the
data received.

## Message Format

The format of a message sent to Sentinel is as follows:

```json
{
  "TimeGenerated": "<server generated>",  // TimeGenerated is the timestamp created by the server upon receiving the alert
  "msg": {}  // StackRox alert object
}
```

### Why Use This Format?

Azure Log Analytics requires a defined table schema for its log storage, which can be cumbersome to maintain given our
data structure. By utilizing the `msg` field as an object, this integration avoids the need for a schema.
The `msg` field is treated as an object in Sentinel, allowing flexibility without enforcing a strict schema.

To properly use this data, users are required to parse the `msg` object within their **Data Collection Rule (DCR)** pipeline.
This parsing step is necessary due to the nature of the data, which often consists of complex, nested JSON objects (e.g., details regarding violations or deployment information). Even if a schema were used, parsing would still be required to extract relevant information from these nested structures.

### Authentication

To generate client certificate and a private key use: `openssl req -x509 -newkey rsa:2048 -days 365 -keyout ca-key.pem -out ca-cert.pem -nodes`

#### Service Principal

To create a new service principal for a resource group to ingest new logs, run:

```
$ az ad sp create-for-rbac --role="Monitoring Metrics Publisher" --scopes="/subscriptions/<subscription-id>/resourceGroups/<resource-group>"
```