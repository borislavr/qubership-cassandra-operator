[[_TOC_]]

# Cassandra Reaper

## Repository structure

* `./docs` - directory with API description.
* `.gitlab-ci.yml` - the CI/CD pipelines configuration.
* `./build.sh` - the entrypoint for build job, it starts docker image build.
* `./description.yaml` - descibes buld sructure of Cassandra Reaper docker image.
* `./Dockerfile` - the Dockerfile for Cassandra Reaper docker image.
* `./run.sh` - the run.sh script used as an entry point.

## Artifacts described

The delivery of this repository contains the following artifact:

* **Cassandra Reaper image**, for instance `artifactorycn.netcracker.com:17008/product/prod.platform.databases_cassandra-reaper:feature_cpdev-105685_tls_20241108-015234` - the image to be deployed to Kubernetes/Openshift, contains the  Adapter binary. Usually we do not need to provide it separately, but for development purpose it might be convinient to update only the image of  adapter deployment in Kubernetes/Openshift to test changes 

## How to start

### Build On-commit

The Cassandra Reaper image is built automatically by CI pipelines on each commit

To get the Cassandra Reaper image:
* Navigate to the Pipeline page of Gitlab and find your pipeline
* Find and open the stage Docker Image
* In the bottom of the job log find the Docker image address

![Docker image of Cassandra Reaper](/docs/internal/images/pipeline_docker_image.jpg)


### Build Manually

Below the manual steps to build Cassandra Reaper image is described

#### Cassandra Reaper Image

1. Go to [DP-Builder](https://cisrvrecn.netcracker.com/job/DP.Pub.Microservice_builder_v2/)
2. Go to the "Build with parameters" tab.
3. Specify:

   * REPOSITORY_NAME - `PROD.Platform.Databases/cassandra-reaper`.
   * LOCATION - your dev branch or `master`.
   
4. Click “Build” button.
5. Find your running build in the “Build History” tab in the DP-Builder page.
6. Wait for the job to finish.
7. Navigate to the job Report section and find the docker image address

![Docker image](/docs/internal/images/job_docker.jpg)


#### Definition of Done

The changes might be marked as fully done if it accepts the following criteria:

1. The ticket's task done.
2. The solution is deployed to dev environment, where it can be tested.
3. Created merge request has:
   1. "Green" pipeline (linter, build, deploy & test jobs are passed).
   2. The title should follow the naming conversation: `<TMS-TICKET-ID>: <CHANGES-SHORT-DESCRIPTION>`.
   3. The description is **fully** filled.

### Deploy to k8s

The Cassandra  Adapter is deployed to kubernetes by [Cassandra Services](https://git.netcracker.com/PROD.Platform.Databases/Ccssandra-services/-/tree/doc/CPDEV-107506_readme/#deploy-to-k8s)


### How to Promote a Release

A promoted artifact is a finalized release that is ready for delivery to customers. Follow these steps to ensure a successful promotion:

1. **Verify Merge Requests**: Ensure that all relevant merge requests (MRs) are merged and have both `milestone` and `labels` set:
   - The **milestone** is crucial as it defines which MRs will be included in the release notes. For instance, if you set the milestone to `0.10.1` for specific MRs and create a tag `0.10.1`, details of those MRs will be listed in the release notes for the `0.10.1` release.
   - **Labels** determine the section in which an MR will appear in the release notes, such as `Documentation`, `Features`, or `Bug fix`.

2. **Create a Tag**: Tag the branch to be released, typically `master`.

3. **Automatic Promotion**: The CI pipeline will promote the release and automatically update the release notes.

### How to troubleshoot

There are no well-defined rules for troubleshooting, as each task is unique, but there are some tips that can do:

* Deploy parameters.
* Logs from the pod.


## CI/CD

The main CI/CD pipeline is designed to automize all basic developer routines.

- test - the stage runs go unit tests.
- buildDocker - the stage builds docker image of the  Adapter.
- dockerValidation - the stage validates that the image can be promoted.
- promoteImage - the stage promotes the image.
- releaseNotes - the stage generates Release Notes.
- milestones - the stage closes current milstone and creates new one.


## Evergreen strategy

To keep the component up to date, the following activities should be performed regularly:

* Vulnerabilities fixing.
* Bug-fixing, improvement and feature implementation.
