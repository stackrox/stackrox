import common.Constants
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import util.ApplicationHealth
import util.Env

OrchestratorMain client = OrchestratorType.create(
        Env.mustGetOrchestratorType(),
        Constants.ORCHESTRATOR_NAMESPACE
)

ApplicationHealth ah = new ApplicationHealth(client, 600)

ah.waitForSensorHealthiness()
ah.waitForCollectorHealthiness()
