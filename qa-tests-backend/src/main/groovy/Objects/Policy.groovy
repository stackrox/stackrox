package Objects

/**
 * Created by parulshukla on 5/31/18.
 */
class Policy {
    def id
    def name
    def severity
    def description
    def rationale
    def remediation
    def disabled
    List<String> categories = new ArrayList<>()
    List<String> scope = new ArrayList<>()
    def enforcement
    List<String> notifiers = new ArrayList<>()
    def privilegePolicy
    def configurationPolicy
}

class PolicyResults{
    List<Policy> policies
}

class AlertByPolicy {
    Policy policy
    def numAlerts
}

class AlertsByPolicy{
   List<AlertByPolicy> alertsByPolicies

}
