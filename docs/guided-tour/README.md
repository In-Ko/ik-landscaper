# Guided Tour

In this tour, you will learn about the different Landscaper features by simple examples. 

## Prerequisites and Basic Definitions

- For all examples, you need a [running Landscaper instance](../gettingstarted/install-landscaper-controller.md).

- A convenient tool we will often use in the following examples is the 
[Landscaper CLI](https://github.com/gardener/landscapercli). 

- During the following exercises, you might need to change files, provided with the examples. For this, you should simply clone this repository and do the required changes on your local files. You could also fork the repo and work on your fork.

- In all examples, 3 Kubernetes clusters are involved:

  - the **Landscaper Host Cluster**, on which the Landscaper runs
  - the **target cluster**, on which the deployments will be done
  - the **Landscaper Resource Cluster**, on which the various custom resources are stored. These custom resources are watched by the Landscaper, and define which deployments should happen on which target cluster.

  It is possible that some or all of these clusters coincide, e.g. in the most simplistic approach, you have only one cluster. Such a "one-cluster-setup" is the easiest way to start working with the Landscaper.

## A Hello World Example

[1. Hello World Example](./hello-world)

## Basics

[2. Upgrading the Hello World Example](./basics/upgrade)

[3. Manifest Deployer Example](./basics/manifest-deployer)

[4. Multiple Deployments in One Installation](./basics/multiple-deployitems)

## Recovering from Errors

[5. Handling an Immediate Error](./error-handling/immediate-error)

[6. Handling a Timeout Error](./error-handling/timeout-error)

[7. Handling a Delete Error](./error-handling/delete-error)

## Blueprints and Components

[8. An Installation with an Externally Stored Blueprint](./blueprints/external-blueprint)

[9. Helm Chart Resources in the Component Descriptor](./blueprints/helm-chart-resource)

[10. Echo Server Example](./blueprints/echo-server)

## Imports and Exports

[11. Import Parameters](./import-export/import-parameters)

[12. Import Data Mappings](./import-export/import-data-mappings)

[13. Export Parameters](./import-export/export-parameters)


<!--
Delete without uninstall
automatic reconcile
reconcile updateOnChangeOnly
Reuse scenario: deploy a blueprint to multiple targets (or target list)
Reuse scenario: upgrade of the component in several Installations
Pull secrets for helm chart repo (with and without secret ref)
Pull secret in context to access a protected oci registry
Helm chart in a private OCI registry, and the difference to a private Helm chart repo
Component descriptor in a private registry
Component descriptor: explain where the path segment "component-descriptor" comes from
Timeouts
Import, export
Subinstallations: with subinstallation from separate components and subinstallations with several blueprints in one component
deploy executions in files
images listed in a component descriptor
additional files in blueprint, e.g. for config data
component descriptor containing a local helm chart

Make use of temp files in the scripts that upload a component descriptor
-->
