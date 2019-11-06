package kanarystatefulset

import (
	"context"
	"os"
	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	kuriseclient "github.com/openkruise/kruise/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kanaryv1alpha1 "github.com/k8s-kanary/kanary/pkg/apis/kanary/v1alpha1"
	"github.com/k8s-kanary/kanary/pkg/config"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/strategies"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils"
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset/utils/enqueue"
	"fmt"
)

var log = logf.Log.WithName("controller_kanarystatefulset")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KanaryStatefulset Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKanaryStatefulset{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kanarystatefulset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KanaryStatefulset
	err = c.Watch(&source.Kind{Type: &kanaryv1alpha1.KanaryStatefulset{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pod and requeue the owner KanaryStatefulset
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &enqueue.RequestForKanaryLabel{})
	return err
}

var _ reconcile.Reconciler = &ReconcileKanaryStatefulset{}
var subResourceDisabled = os.Getenv(config.KanaryStatusSubresourceDisabledEnvVar) == "1"

// ReconcileKanaryStatefulset reconciles a KanaryStatefulset object
type ReconcileKanaryStatefulset struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KanaryStatefulset object and makes changes based on the state read
// and what is in the KanaryStatefulset.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKanaryStatefulset) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Namespace", request.Namespace, "KanaryStatefulset", request.Name)
	reqLogger.Info("Reconciling KanaryStatefulset")

	// Fetch the KanaryStatefulset instance
	instance := &kanaryv1alpha1.KanaryStatefulset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// kanary 要更新的 spec 跟原来的 spec 相同
	if !kanaryv1alpha1.IsDefaultedKanaryStatefulset(instance) {
		reqLogger.Info("Defaulting values")
		defaultedInstance := kanaryv1alpha1.DefaultKanaryStatefulset(instance)
		err = r.client.Update(context.TODO(), defaultedInstance)
		if err != nil {
			reqLogger.Error(err, "failed to update KanaryStatefulset")
			return reconcile.Result{}, err
		}
		reqLogger.Info("Defaulting done, requeuing")
		// KanaryStatefulset is now defaulted return and requeue
		return reconcile.Result{Requeue: true}, nil
	}
	
	// scheduled => Running => Succeed => Scheduled  => canaryDeployed
	
	// var deployment, canarydeployment *appsv1beta1.Deployment
	// var statefulset, canaryStatefulset *appsv1beta1.StatefulSet
	
	statefulset, needsReturn, result, err := r.getStatefulSet(reqLogger, instance)
	
	// 拿到了 statuflset instance.
	if needsReturn {
		return updateKanaryStatefulsetStatus(r.client, reqLogger, instance, metav1.Now(), result, err)
	}
	
	//Check scheduling
	reqLogger.Info("Scheduling")
	if newstatus, schedResult := strategies.ApplyScheduling(reqLogger, instance); newstatus != nil || schedResult != nil {
		return utils.UpdateKanaryStatefulsetStatus(r.client, subResourceDisabled, reqLogger, instance, newstatus, *schedResult, nil)
	}
	
	fmt.Println(statefulset)
	return reconcile.Result{}, nil
	
	/*
	canarydeployment, needsReturn, result, err = r.manageCanaryDeploymentCreation(reqLogger, instance, utils.GetCanaryDeploymentName(instance))
	if needsReturn {
		return updateKanaryStatefulsetStatus(r.client, reqLogger, instance, metav1.Now(), result, err)
	}

	strategy, err := strategies.NewStrategy(&instance.Spec)
	if err != nil {
		reqLogger.Error(err, "failed to instance the KanaryStatefulset strategies")
		return reconcile.Result{}, err
	}
	if strategy == nil {
		return updateKanaryStatefulsetStatus(r.client, reqLogger, instance, metav1.Now(), result, err)
	}
	
	reqLogger.Info("Applying")
	return strategy.Apply(r.client, reqLogger, instance, deployment, canarydeployment, statefulset)
	*/
}

/*
func (r *ReconcileKanaryStatefulset) manageCanaryStatefulsetCreation(reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, name string) (*appsv1beta1.Deployment, bool, reconcile.Result, error) {
	// check that the deployment template was not updated since the creation
	currentHash, err := comparison.GenerateMD5DeploymentSpec(&kd.Spec.Template.Spec)
	if err != nil {
		reqLogger.Error(err, "failed to generate Deployment template MD5")
		return nil, true, reconcile.Result{}, err
	}

	deployment := &appsv1beta1.Deployment{}
	result := reconcile.Result{}
	
	if kd.Spec.StatefulSetName != "" {
		sts, err := kuriseclient.GetGenericClient().KruiseClient.AppsV1alpha1().StatefulSets(kd.Namespace).Get(kd.Spec.StatefulSetName, metav1.GetOptions{})
		if err != nil {
			reqLogger.Error(err, "failed to get statefulset")
			return deployment, true, reconcile.Result{}, err
		}
		updateSts := sts.DeepCopy()
		if kd.Spec.Scale.Static == nil {
			reqLogger.Error(err, "only support static scale")
			return deployment, true, reconcile.Result{}, err
		}
		updateSts.Spec.UpdateStrategy.RollingUpdate.Partition = kd.Spec.Scale.Static.Replicas
		updateSts.Spec.Template = kd.Spec.Template.Spec.Template
		_, err = kuriseclient.GetGenericClient().KruiseClient.AppsV1alpha1().StatefulSets(sts.Namespace).Update(updateSts)
		if err != nil {
			reqLogger.Error(err, "failed to update Deployment replicas", "Namespace", updateSts.Namespace, "Deployment", updateSts.Name)
		}
		
		return deployment, false, reconcile.Result{}, err
	}
	
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: kd.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment, err = utils.NewCanaryDeploymentFromKanaryStatefulsetTemplate(r.client, kd, r.scheme, false)
		if err != nil {
			reqLogger.Error(err, "failed to create the Deployment artifact")
			return deployment, true, result, err
		}

		reqLogger.Info("Creating a new Deployment")
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "failed to create new Deployment")
			return deployment, true, result, err
		}
		newStatus := kd.Status.DeepCopy()
		newStatus.CurrentHash = currentHash
		utils.UpdateKanaryStatefulsetStatusCondition(newStatus, metav1.Now(), kanaryv1alpha1.ActivatedKanaryStatefulsetConditionType, corev1.ConditionTrue, "", false)
		result.Requeue = true
		result, err = utils.UpdateKanaryStatefulsetStatus(r.client, subResourceDisabled, reqLogger, kd, newStatus, result, err)
		// Deployment created successfully - return and requeue
		return deployment, true, result, err
	} else if err != nil {
		reqLogger.Error(err, "failed to get Deployment")
		return deployment, true, reconcile.Result{}, err
	}

	if kd.Status.CurrentHash != "" && kd.Status.CurrentHash != currentHash {
		err = r.client.Delete(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "failed to delete deprecated Deployment")
			return deployment, true, reconcile.Result{RequeueAfter: time.Second}, err
		}
	}

	return deployment, false, reconcile.Result{}, err
}

func (r *ReconcileKanaryStatefulset) manageDeploymentCreationFunc(reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, name string, createFunc func(*kanaryv1alpha1.KanaryStatefulset, *runtime.Scheme, bool) (*appsv1beta1.Deployment, error)) (*appsv1beta1.Deployment, bool, reconcile.Result, error) {
	deployment := &appsv1beta1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: kd.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		deployment, err = createFunc(kd, r.scheme, false)
		if err != nil {
			reqLogger.Error(err, "failed to create the Deployment artifact")
			return deployment, true, reconcile.Result{}, err
		}
	
		reqLogger.Info("Creating a new Deployment")
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "failed to create new Deployment")
			return deployment, true, reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return deployment, true, reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get Deployment")
		return deployment, true, reconcile.Result{}, err
	}

	return deployment, false, reconcile.Result{}, err
}
*/

func updateKanaryStatefulsetStatus(kclient client.Client, reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset, now metav1.Time, result reconcile.Result, err error) (reconcile.Result, error) {
	newStatus := kd.Status.DeepCopy()
	utils.UpdateKanaryStatefulsetStatusConditionsFailure(newStatus, now, err)
	return utils.UpdateKanaryStatefulsetStatus(kclient, subResourceDisabled, reqLogger, kd, newStatus, result, err)
}


func (r *ReconcileKanaryStatefulset) getStatefulSet(reqLogger logr.Logger, kd *kanaryv1alpha1.KanaryStatefulset) (*kruisev1alpha1.StatefulSet, bool, reconcile.Result, error) {
	statefulset, err := kuriseclient.GetGenericClient().KruiseClient.AppsV1alpha1().StatefulSets(kd.Namespace).Get(kd.Spec.StatefulSetName, )
	if err != nil {
		reqLogger.Error(err, "failed to get statefulset")
		return &kruisev1alpha1.StatefulSet{}, true, reconcile.Result{}, err
	}
	return statefulset, false, reconcile.Result{}, err
}