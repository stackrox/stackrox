# Microsoft Sentinel Integration

In the `./data/` directory, there is a file named `example-alert.json` that contains an example of an alert sent
to Microsoft Sentinel.

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