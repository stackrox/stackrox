package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ExternalBackupServiceGrpc
import io.stackrox.proto.storage.ExternalBackupOuterClass
import io.stackrox.proto.storage.ScheduleOuterClass
import util.Env

@Slf4j
class ExternalBackupService extends BaseService {
    static getExternalBackupClient() {
        return ExternalBackupServiceGrpc.newBlockingStub(getChannel())
    }

    static testExternalBackup(ExternalBackupOuterClass.ExternalBackup backup) {
        try {
            getExternalBackupClient().testExternalBackup(backup)
            return true
        } catch (Exception e) {
            log.warn("test external backup failed", e)
            return false
        }
    }

    static ExternalBackupOuterClass.ExternalBackup getS3IntegrationConfig(
            String name,
            String bucket = Env.mustGetAWSS3BucketName(),
            String region = Env.mustGetAWSS3BucketRegion(),
            String endpoint = "",
            String accessKeyId = Env.mustGetAWSAccessKeyID(),
            String accessKey = Env.mustGetAWSSecretAccessKey())  {
        ExternalBackupOuterClass.S3Config s3Config =  ExternalBackupOuterClass.S3Config.newBuilder()
                .setObjectPrefix(UUID.randomUUID().toString())
                .setBucket(bucket)
                .setRegion(region)
                .setEndpoint(endpoint)
                .setUseIam(false)
                .setAccessKeyId(accessKeyId)
                .setSecretAccessKey(accessKey)
                .build()

        return ExternalBackupOuterClass.ExternalBackup.newBuilder()
                .setName(name)
                .setType("s3")
                .setBackupsToKeep(1)
                .setSchedule(ScheduleOuterClass.Schedule.newBuilder()
                        .setIntervalType(ScheduleOuterClass.Schedule.IntervalType.DAILY)
                        .setHour(0) //12:00 AM
                        .build()
                )
                .setS3(s3Config)
                .build()
    }
}
