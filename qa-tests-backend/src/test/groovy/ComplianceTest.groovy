import groups.BAT
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.ComplianceRunScheduleInfo
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ComplianceManagementService
import services.ComplianceService
import spock.lang.Unroll
import v1.ComplianceServiceOuterClass.ComplianceStandardMetadata

class ComplianceTest extends BaseSpecification {
    @Unroll
    @Category(BAT)
    def "Verify that we can run a benchmark: "(String benchmarkName) {
        when:
        "Trigger a compliance benchmark"
        String benchmarkID = ComplianceService.getBenchmark(benchmarkName)
        println ("Found benchmark ID ${benchmarkID} for ${benchmarkName}")
        String clusterID = ClusterService.getClusterId()
        ComplianceService.runBenchmark(benchmarkID, clusterID)

        then:
        "Verify Scan is run"
        assert ComplianceService.checkBenchmarkRan(benchmarkID, clusterID)

        cleanup:
        "Make sure the daemonset benchmark is gone"
        orchestrator.waitForDaemonSetDeletion("benchmark", "stackrox")

        where:
        "Data inputs are :"

        benchmarkName | _
        "CIS Kubernetes v1.2.0 Benchmark" | _
        "CIS Docker v1.1.0 Benchmark" | _
    }

    def "Verify compliance scheduling"() {
        given:
        "ignore test - compliance not enabled by default"
        Assume.assumeTrue(false)

        and:
        "List of Standards"
        List<ComplianceStandardMetadata> standards = ComplianceService.getComplianceStandards()

        when:
        "create a schedule"
        ComplianceRunScheduleInfo info = ComplianceManagementService.addSchedule(
                        standards.get(0).id,
                        ClusterService.getClusterId(),
                        "* 4 * * *"
                )
        assert info
        assert ComplianceManagementService.getSchedules().find { it.schedule.id == info.schedule.id }

        and:
        "verify schedule details"
        Calendar nextRun = Calendar.getInstance(TimeZone.getTimeZone("GMT"))
        nextRun.setTime(new Date(info.nextRunTime.seconds * 1000))
        Calendar now = Calendar.getInstance(TimeZone.getTimeZone("GMT"))
        now.get(Calendar.HOUR_OF_DAY) < 4 ?: now.add(Calendar.DAY_OF_YEAR, 1)
        assert nextRun.get(Calendar.HOUR_OF_DAY) == 4
        assert nextRun.get(Calendar.DAY_OF_YEAR) == now.get(Calendar.DAY_OF_YEAR)

        and:
        "update schedule"
        int minute = now.get(Calendar.MINUTE)
        int hour = now.get(Calendar.HOUR_OF_DAY)
        if (minute < 59) {
            minute++
        } else {
            minute = 0
            hour ++
        }
        String cron = "${minute} ${hour} * * *"
        ComplianceRunScheduleInfo update = ComplianceManagementService.updateSchedule(
                        info.schedule.id,
                        standards.get(0).id,
                        ClusterService.getClusterId(),
                        cron
                )
        assert update

        and:
        "verify update"
        assert ComplianceManagementService.getSchedules().find {
            it.schedule.id == info.schedule.id && it.schedule.crontabSpec == cron
        }

        and:
        "verify standard started on schedule"
        println "Waiting for schedule to start..."
        while (now.get(Calendar.MINUTE) < minute) {
            sleep 1000
            now = Calendar.getInstance(TimeZone.getTimeZone("GMT"))
        }
        long mostRecent = 0
        ComplianceManagementService.getRecentRuns(standards.get(0).id).each {
            if (it.startTime.seconds > mostRecent) {
                mostRecent = it.startTime.seconds
            }
        }
        assert mostRecent >= update.nextRunTime.seconds

        then:
        "delete schedule"
        ComplianceManagementService.deleteSchedule(info.schedule.id)
    }

}
