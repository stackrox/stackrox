package Objects

import java.util.Map.Entry

import com.google.gson.JsonArray
import com.google.gson.JsonObject

class KubernetesDeployment{
    String apiVersion
    String kindJson
    String deploymentName
    Map<String, String> metaLabels
    int replicasNum
    Map<String, String> templateLabels
    String containersImage
    String containersName
    List<Integer> containersPorts

    KubernetesDeployment() {
        this.apiVersion = "extensions/v1beta1"
        this.kindJson = "Deployment"
        this.deploymentName = "qa-app"
        this.metaLabels = new HashMap<String, String>()
        this.replicasNum = 1
        this.templateLabels =  new HashMap<String, String>()
        this.containersImage = "docker.io/library/nginx:1.7.9"
        this.containersName = "nginx"
        this.containersPorts = new LinkedList<Integer>()

        this.metaLabels.put("app", "test")
        this.templateLabels.put("app", "test")
    }
    String getDeploymentName(){
        return this.deploymentName
    }
    void setDeploymentName(String deploymentName) {
        this.deploymentName = deploymentName
    }
    void addMetaLabels(String labelName, String labelValue) {
        this.metaLabels.put(labelName, labelValue)
    }
    void setReplicasNum(int replicasNum) {
        this.replicasNum = replicasNum
    }
    void addTemplateLabels(String labelName, String labelValue) {
        this.templateLabels.put(labelName, labelValue)
    }
    void setContainersImage(String containersImage) {
        this.containersImage = containersImage
    }
    void setContainersName(String containersName) {
        this.containersName = containersName
    }
    void addContainerPort(int port) {
        this.containersPorts.add(port)
    }

    String generateDeployment() {
        JsonObject jsonDeployment = new JsonObject()
        jsonDeployment.addProperty("apiVersion", apiVersion)
        jsonDeployment.addProperty("kind", kindJson)

        JsonObject metaDataJO = new JsonObject()
        metaDataJO.addProperty("name", deploymentName)
        if (metaLabels.size() != 0) {
            JsonObject metaLabelsJO = new JsonObject()
            Iterator<Entry<String, String>> labelIt = metaLabels.entrySet().iterator()
            while (labelIt.hasNext()) {
                Map.Entry<String, String> pair = (Map.Entry<String, String>)labelIt.next()
                metaLabelsJO.addProperty(pair.getKey(), pair.getValue())
                labelIt.remove()
            }
            metaDataJO.add("labels", metaLabelsJO)
        }
        jsonDeployment.add("metadata", metaDataJO)

        JsonObject specJO = new JsonObject()
        specJO.addProperty("replicas", replicasNum)

        JsonObject templateJO = new JsonObject()

        JsonObject templateMetaDataJO =new JsonObject()
        if (templateLabels.size() != 0) {
            JsonObject templateMetaDataLablesJO = new JsonObject()
            Iterator<Entry<String, String>> labelIt = templateLabels.entrySet().iterator()
            while (labelIt.hasNext()) {
                Map.Entry<String, String> pair = (Map.Entry<String, String>)labelIt.next()
                templateMetaDataLablesJO.addProperty(pair.getKey(), pair.getValue())
                labelIt.remove()
            }
            templateMetaDataJO.add("labels", templateMetaDataLablesJO)
        }
        templateJO.add("metadata", templateMetaDataJO)

        JsonObject templateSpecJO = new JsonObject()
        JsonArray containersJA = new JsonArray()
        JsonObject containerJO = new JsonObject()
        containerJO.addProperty("image", containersImage)
        containerJO.addProperty("name", containersName)
        if (containersPorts.size() != 0) {
            JsonArray portsJA = new JsonArray()
            Iterator<Integer> portIt = containersPorts.iterator()
            while (portIt.hasNext()) {
                int portNum = portIt.next()
                JsonObject portJO = new JsonObject()
                portJO.addProperty("containerPort", portNum)
                portsJA.add(portJO)
            }
            containerJO.add("ports", portsJA)
        }
        containersJA.add(containerJO)
        templateSpecJO.add("containers", containersJA)
        templateJO.add("spec", templateSpecJO)
        specJO.add("template", templateJO)
        jsonDeployment.add("spec", specJO)

        return jsonDeployment.toString()
    }
}
