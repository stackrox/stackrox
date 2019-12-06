package util

import com.google.gson.JsonElement

class RawParser {

    JsonElement id
    JsonElement name
    JsonElement policy
    JsonElement deployment

    class Policy {
        JsonElement name
    }

    class Deployment {
        JsonElement id
        JsonElement name
        JsonElement type
        JsonElement namespace
        JsonElement clusterName
    }
}
