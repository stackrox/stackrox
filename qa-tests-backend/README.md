**Platform Integration Tests**
Framework is designed to Integration and Functional flows through APIs 

**Build Tooling**
Framework uses Gradle as a build tool , Groovy Language and Spock as Framework 
  Gradle : 4.10.1 use Homebrew 'brew install gradle'
  Make : make style 

**How to Add/run tests**
  New tests are added with group Annotations , current CI integration uses BAT annotation which is added under groups. New target can be added to make file before running tests.
   Add values to your ENV  
    export  CLUSTER=K8S
    export HOSTNAME=localhost
    export PORT=${LOCAL_PORT} (8000)
    make -C qa-tests-backend bat-test
  
  Framework used Auto generated protos , to make sure we use latest protos , navigate to qa-tests-backend directory 
  make proto-generated-srcs
  
  Test outputs are integrated with spock-reports plugin. All the reports are added under build/spock-reports folder
   Report is generated with all the tests executed with asserts for the failed and the steps executed 
  
  Running tests from IDE
    Run > Edit Configurations
    Select Groovy , add a new Configuration 
      Script path : github.com/stackrox/rox/qa-tests-backend/src/test/groovy/<Groovy class name>.groovy
      Working Directory : github.com/stackrox/rox/qa-tests-backend
      Environment Variables : CLUSTER=K8S;HOSTNAME=localhost;PORT=8000;
      module : qa-test-backend_test
    Save the configuration and Run the tests  