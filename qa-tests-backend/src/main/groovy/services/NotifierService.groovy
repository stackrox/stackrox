package services

import groovy.json.JsonSlurper
import io.stackrox.proto.api.v1.NotifierServiceGrpc
import io.stackrox.proto.api.v1.NotifierServiceOuterClass
import io.stackrox.proto.storage.NotifierOuterClass
import util.Timer

class NotifierService extends BaseService {
    private static final PAGERDUTY_URL = "https://api.pagerduty.com/incidents?sort_by=created_at%3Adesc&&limit=1"
    private static final PAGERDUTY_TOKEN = "qWT6sXfp_Lvz-pddxcCg"
    private static final PAGERDUTY_API_KEY = "9e2d142a2946419c9192a0b224dd811b"
    static getNotifierClient() {
        return NotifierServiceGrpc.newBlockingStub(getChannel())
    }
    static testNotifier(NotifierOuterClass.Notifier notifier) {
        try {
            getNotifierClient().testNotifier(notifier)
            return true
        } catch (Exception e) {
            println e.toString()
            return false
        }
    }

    static deleteNotifier(String id) {
        try {
            getNotifierClient().deleteNotifier(
                    NotifierServiceOuterClass.DeleteNotifierRequest.newBuilder()
                            .setId(id)
                            .setForce(true)
                            .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static addPagerDutyNotifier(String name) {
        try {
            return getNotifierClient().postNotifier(
                    NotifierOuterClass.Notifier.newBuilder()
                            .setType("pagerduty")
                            .setName(name)
                            .setEnabled(true)
                            .setUiEndpoint("https://localhost:8000")
                            .setPagerduty(NotifierOuterClass.PagerDuty.newBuilder()
                            .setApiKey(PAGERDUTY_API_KEY)
                            )
                            .build()
                    )
        } catch (Exception e) {
            println "Add PagerDuty Service:" + e.toString()
        }
    }

    static getFirstPagerDutyIncident() {
        try {
            def con =(HttpURLConnection) new URL(PAGERDUTY_URL).openConnection()
            con.setRequestMethod("GET")
            con.setRequestProperty("Content-Type", "application/json; charset=UTF-8")
            con.setRequestProperty("Accept", "application/vnd.pagerduty+json;version=2")
            con.setRequestProperty("Authorization", "Token token=${PAGERDUTY_TOKEN}")

            def jsonSlurper = new JsonSlurper()
            return jsonSlurper.parseText(con.getInputStream().getText())
        } catch (Exception e) {
            println "Get PagerDuty first incident:" + e.toString()
        }
    }

    static waitForPagerDutyUpdate(int preNum) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for PagerDuty Update"
            def object = getFirstPagerDutyIncident()
            int curNum = object.incidents[0].incident_number

            if (curNum > preNum) {
                return object
            }
        }
        println "Time out for Waiting for PagerDuty Update"
        return null
    }
}
