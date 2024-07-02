package objects

class Namespace {
    def uid
    def name
    Map<String,String> labels
    List<Tuple> deploymentCount
    def secretsCount
    def networkPolicyCount
}
