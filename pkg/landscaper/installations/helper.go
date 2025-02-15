// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lscheme "github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	lsutils "github.com/gardener/landscaper/pkg/utils"
)

var componentInstallationGVK schema.GroupVersionKind

func init() {
	var err error
	componentInstallationGVK, err = apiutil.GVKForObject(&lsv1alpha1.Installation{}, lscheme.LandscaperScheme)
	runtime.Must(err)
}

// IsRootInstallation returns if the installation is a root element.
func IsRootInstallation(inst *lsv1alpha1.Installation) bool {
	_, isOwned := kubernetes.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return !isOwned
}

// GetParentInstallationName returns the name of parent installation that encompasses the given installation.
func GetParentInstallationName(inst *lsv1alpha1.Installation) string {
	name, _ := kubernetes.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return name
}

// CreateInternalInstallationBases creates internal installation bases for a list of ComponentInstallations
func CreateInternalInstallationBases(installations ...*lsv1alpha1.Installation) []*InstallationAndImports {
	if len(installations) == 0 {
		return nil
	}
	internalInstallations := make([]*InstallationAndImports, len(installations))
	for i, inst := range installations {
		inInst := CreateInternalInstallationBase(inst)
		internalInstallations[i] = inInst
	}
	return internalInstallations
}

// ResolveComponentDescriptor resolves the component descriptor of a installation.
// Inline Component Descriptors take precedence
func ResolveComponentDescriptor(ctx context.Context, compRepo ctf.ComponentResolver, inst *lsv1alpha1.Installation, overwriter componentoverwrites.Overwriter) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	if inst.Spec.ComponentDescriptor == nil || (inst.Spec.ComponentDescriptor.Reference == nil && inst.Spec.ComponentDescriptor.Inline == nil) {
		return nil, nil, nil
	}
	var (
		repoCtx *cdv2.UnstructuredTypedObject
		ref     cdv2.ObjectMeta
	)
	//case inline component descriptor
	if inst.Spec.ComponentDescriptor.Inline != nil {
		repoCtx = inst.Spec.ComponentDescriptor.Inline.GetEffectiveRepositoryContext()
		ref = inst.Spec.ComponentDescriptor.Inline.ObjectMeta
	} else if inst.Spec.ComponentDescriptor.Reference != nil {
		// case remote reference
		repoCtx = inst.Spec.ComponentDescriptor.Reference.RepositoryContext
		ref = inst.Spec.ComponentDescriptor.Reference.ObjectMeta()
	}
	return cdutils.ResolveWithBlobResolverWithOverwriter(ctx, compRepo, repoCtx, ref.GetName(), ref.GetVersion(), overwriter)
}

// CreateInternalInstallation creates an internal installation for an Installation
// DEPRECATED: use CreateInternalInstallationWithContext instead
func CreateInternalInstallation(ctx context.Context, compResolver ctf.ComponentResolver, inst *lsv1alpha1.Installation) (*InstallationImportsAndBlueprint, error) {
	if inst == nil {
		return nil, nil
	}
	cdRef := GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	blue, err := blueprints.Resolve(ctx, compResolver, cdRef, inst.Spec.Blueprint)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return NewInstallationImportsAndBlueprint(inst, blue), nil
}

// CreateInternalInstallationWithContext creates an internal installation for an Installation
func CreateInternalInstallationWithContext(ctx context.Context,
	inst *lsv1alpha1.Installation,
	kubeClient client.Client,
	compResolver ctf.ComponentResolver) (*InstallationImportsAndBlueprint, error) {
	if inst == nil {
		return nil, nil
	}
	lsCtx, err := GetExternalContext(ctx, kubeClient, inst)
	if err != nil {
		return nil, err
	}
	blue, err := blueprints.Resolve(ctx, compResolver, lsCtx.ComponentDescriptorRef(), inst.Spec.Blueprint)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return NewInstallationImportsAndBlueprint(inst, blue), nil
}

// CreateInternalInstallationBase creates an internal installation base for an Installation
func CreateInternalInstallationBase(inst *lsv1alpha1.Installation) *InstallationAndImports {
	if inst == nil {
		return nil
	}
	return NewInstallationAndImports(inst)
}

// GetDataImport fetches the data import from the cluster.
func GetDataImport(ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *InstallationAndImports,
	dataImport lsv1alpha1.DataImport) (*dataobjects.DataObject, *metav1.OwnerReference, error) {

	var rawDataObject *lsv1alpha1.DataObject
	// get deploy item from current context
	if len(dataImport.DataRef) != 0 {
		rawDataObject = &lsv1alpha1.DataObject{}
		doName := lsv1alpha1helper.GenerateDataObjectName(contextName, dataImport.DataRef)
		if err := kubeClient.Get(ctx, kubernetes.ObjectKey(doName, inst.GetInstallation().Namespace), rawDataObject); err != nil {
			return nil, nil, fmt.Errorf("unable to fetch data object %s (%s/%s): %w", doName, contextName, dataImport.DataRef, err)
		}
	}
	if dataImport.SecretRef != nil {
		_, data, gen, err := lsutils.ResolveSecretReference(ctx, kubeClient, dataImport.SecretRef)
		if err != nil {
			return nil, nil, err
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data.RawMessage = data
		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(gen)
	}
	if dataImport.ConfigMapRef != nil {
		_, data, gen, err := lsutils.ResolveConfigMapReference(ctx, kubeClient, dataImport.ConfigMapRef)
		if err != nil {
			return nil, nil, err
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data.RawMessage = data
		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(gen)
	}

	do, err := dataobjects.NewFromDataObject(rawDataObject)
	if err != nil {
		return nil, nil, err
	}
	do.Def = &dataImport

	owner := kubernetes.GetOwner(do.Raw.ObjectMeta)
	return do, owner, nil
}

// GetTargets returns all targets and import references defined by a target import.
func GetTargets(ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *lsv1alpha1.Installation,
	targetImport lsv1alpha1.TargetImport) ([]*dataobjects.TargetExtension, []string, error) {
	var targets []*dataobjects.TargetExtension
	var targetImportReferences []string
	if len(targetImport.Target) != 0 {
		target, err := GetTargetImport(ctx, kubeClient, contextName, inst, targetImport)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: unable to get target for '%s': %w", targetImport.Name, targetImport.Name, err)
		}
		targets = []*dataobjects.TargetExtension{target}
		targetImportReferences = []string{targetImport.Target}
	} else if targetImport.Targets != nil {
		tl, err := GetTargetListImportByNames(ctx, kubeClient, contextName, inst, targetImport)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: unable to get targetlist for '%s': %w", targetImport.Name, targetImport.Name, err)
		}
		if len(tl.GetTargetExtensions()) != len(targetImport.Targets) {
			return nil, nil, fmt.Errorf("%s: targetlist size mismatch: %d targets were expected but %d were fetched from the cluster",
				targetImport.Name, len(targetImport.Targets), len(tl.GetTargetExtensions()))
		}
		targets = tl.GetTargetExtensions()
		targetImportReferences = targetImport.Targets
	} else if len(targetImport.TargetListReference) != 0 {
		tl, err := GetTargetListImportBySelector(ctx, kubeClient, contextName, inst, map[string]string{lsv1alpha1.DataObjectKeyLabel: targetImport.TargetListReference}, targetImport, true)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: unable to get targetlist for '%s': %w", targetImport.Name, targetImport.Name, err)
		}
		targets = tl.GetTargetExtensions()
		targetImportReferences = []string{targetImport.TargetListReference}
	} else {
		return nil, nil, fmt.Errorf("invalid target import '%s': one of target, targets, or targetListRef must be specified", targetImport.Name)
	}
	return targets, targetImportReferences, nil
}

// GetTargetImport fetches the target import from the cluster.
func GetTargetImport(ctx context.Context, kubeClient client.Client, contextName string, inst *lsv1alpha1.Installation, targetImport lsv1alpha1.TargetImport) (*dataobjects.TargetExtension, error) {
	targetName := targetImport.Target
	target := &lsv1alpha1.Target{}
	targetName = lsv1alpha1helper.GenerateDataObjectName(contextName, targetName)
	if err := kubeClient.Get(ctx, kubernetes.ObjectKey(targetName, inst.Namespace), target); err != nil {
		return nil, err
	}

	targetExtension := dataobjects.NewTargetExtension(target, &targetImport)

	return targetExtension, nil
}

// GetTargetListImportByNames fetches the target imports from the cluster, based on a list of target names.
func GetTargetListImportByNames(
	ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *lsv1alpha1.Installation,
	targetImport lsv1alpha1.TargetImport) (*dataobjects.TargetExtensionList, error) {
	targets := make([]lsv1alpha1.Target, len(targetImport.Targets))
	for i, targetName := range targetImport.Targets {
		// get deploy item from current context
		raw := &lsv1alpha1.Target{}
		targetName = lsv1alpha1helper.GenerateDataObjectName(contextName, targetName)
		if err := kubeClient.Get(ctx, kubernetes.ObjectKey(targetName, inst.Namespace), raw); err != nil {
			return nil, err
		}
		targets[i] = *raw
	}
	targetExtensionList := dataobjects.NewTargetExtensionList(targets, &targetImport)

	return targetExtensionList, nil
}

// GetTargetListImportBySelector fetches the target imports from the cluster, based on a label selector.
// If restrictToImport is true, a label selector will be added which fetches only targets that are marked as import.
func GetTargetListImportBySelector(
	ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *lsv1alpha1.Installation,
	selector map[string]string,
	targetImport lsv1alpha1.TargetImport,
	restrictToImport bool) (*dataobjects.TargetExtensionList, error) {
	targets := &lsv1alpha1.TargetList{}
	// construct label selector
	contextSelector := labels.NewSelector()
	if len(contextName) != 0 {
		// top-level targets probably don't have an empty context set, so only add the selector if there actually is a context
		r, err := labels.NewRequirement(lsv1alpha1.DataObjectContextLabel, selection.Equals, []string{contextName})
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	} else {
		// top-level targets probably don't have an empty context set, so check for non-existence of the label
		r, err := labels.NewRequirement(lsv1alpha1.DataObjectContextLabel, selection.DoesNotExist, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	}
	// add given labels to selector
	for k, v := range selector {
		r, err := labels.NewRequirement(k, selection.Equals, []string{v})
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	}
	if restrictToImport {
		// add further labels to ensure that only targets imported by that installation are selected
		r, err := labels.NewRequirement(lsv1alpha1.DataObjectSourceTypeLabel, selection.Equals, []string{string(lsv1alpha1.ImportDataObjectSourceType)})
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	}
	if err := kubeClient.List(ctx, targets, client.InNamespace(inst.Namespace), &client.ListOptions{LabelSelector: contextSelector}); err != nil {
		return nil, err
	}
	targetExtensionList := dataobjects.NewTargetExtensionList(targets.Items, &targetImport)
	return targetExtensionList, nil
}

// GetReferenceFromComponentDescriptorDefinition tries to extract a component descriptor reference from a given component descriptor definition
func GetReferenceFromComponentDescriptorDefinition(cdDef *lsv1alpha1.ComponentDescriptorDefinition) *lsv1alpha1.ComponentDescriptorReference {
	if cdDef == nil {
		return nil
	}

	if cdDef.Inline != nil {
		repoCtx := cdDef.Inline.GetEffectiveRepositoryContext()
		return &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: repoCtx,
			ComponentName:     cdDef.Inline.Name,
			Version:           cdDef.Inline.Version,
		}
	}

	return cdDef.Reference
}

func OwnerReferenceIsInstallation(owner *metav1.OwnerReference) bool {
	return owner != nil && owner.Kind == "Installation"
}

func OwnerReferenceIsInstallationButNoParent(owner *metav1.OwnerReference, installation *lsv1alpha1.Installation) bool {
	parentName := GetParentInstallationName(installation)

	return OwnerReferenceIsInstallation(owner) && owner.Name != parentName

}
