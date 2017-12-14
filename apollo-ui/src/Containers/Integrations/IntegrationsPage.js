import React, { Component } from "react";
import aqua from "images/aqua.svg";
import artifactory from "images/artifactory.svg";
import azure from "images/azure.svg";
import dockerEnt from "images/docker-ent.svg";
import docker from "images/docker.svg";
import google from "images/google.svg";
import grafeas from "images/grafeas.svg";
import jira from "images/jira.svg";
import kubernetes from "images/kubernetes.svg";
import openshift from "images/openshift.svg";
import pagerduty from "images/pagerduty.svg";
import slack from "images/slack.svg";
import twistlock from "images/twistlock.svg";


class IntegrationsContainer extends Component {
  render() {
    return <section className="flex mt-4">
        <div class="md:w-2/3">
          <h1 class="font-500 mx-3">Data sources</h1>
          <div>
            <h2 className="mx-3 mt-5 mb-2 font-500">Registries</h2>
            <div className="flex flex-wrap">
              <div className="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow text-center ">
                  <img className="w-32 h-32 mb-4" src={docker} alt="Docker Trusted Registry" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Docker Trusted Registry
                  </div>
                </div>
              </div>

              <div class="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={docker} alt="Docker Hub" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Docker Hub
                  </div>
                </div>
              </div>

              <div class="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={openshift} alt="Openshift" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Openshift Container Registry
                  </div>
                </div>
              </div>

              <div class="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={google} alt="Google Container Registry" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Google Container Registry
                  </div>
                </div>
              </div>

              <div class="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={azure} alt="Azure Container Registry" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Azure Container Registry
                  </div>
                </div>
              </div>

              <div class="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={artifactory} alt="Artifactory" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Artifactory
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div>
            <h2 className="mx-3 mt-5 mb-2 font-500">
              Orchestrators & Container Platforms
            </h2>
            <div className="flex">
              <div className="p-3 flex-1">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={kubernetes} alt="Kubernetes" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Kubernetes
                  </div>
                </div>
              </div>

              <div class="p-3 flex-1">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={docker} alt="Docker Swarm" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Docker Swarm
                  </div>
                </div>
              </div>

              <div class="p-3 flex-1">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={openshift} alt="Openshift" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Red Hat Openshift
                  </div>
                </div>
              </div>

              <div class="p-3 flex-1">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={dockerEnt} alt="Docker Enterprise Edition" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Docker Enterprise Edition
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="mb-6">
            <h2 className="mx-3 mt-5 mb-2 font-500">Scanning & Governance Tools</h2>
            <div className="flex flex-wrap">
              <div className="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={twistlock} alt="Twistlock" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    TwistLock
                  </div>
                </div>
              </div>

              <div class="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={aqua} alt="Aqua" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Aqua
                  </div>
                </div>
              </div>

              <div class="p-3 w-1/3">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={grafeas} alt="Grafeas" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Grafeas
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div />
        </div>
        <aside className="flex-col h-full bg-primary-100 md:w-1/3 border-l border-primary-300 ">
            <h1 class="font-500 mx-3">Workflow plugins</h1>
          <div>
            <div className="flex flex-wrap">
              <div className="p-3 w-full">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={slack} alt="Slack" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Slack
                  </div>
                </div>
              </div>

              <div class="p-3 w-full">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={pagerduty} alt="Pagerduty" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Pagerduty
                  </div>
                </div>
              </div>

              <div class="p-3 w-full">
                <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                  <img className="w-32 h-32 mb-4" src={jira} alt="Jira" />
                  <div className="font-bold text-xl pt-4 mb-2 border-t border-base-200">
                    Jira
                  </div>
                </div>
              </div>
            </div>
          </div>
        </aside>
      </section>;
  }
}

export default IntegrationsContainer;
