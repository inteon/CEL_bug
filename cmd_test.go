package cmd_test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	_, b, _, _  = goruntime.Caller(0)
	projectRoot = filepath.Join(filepath.Dir(b), ".")
)

func TestBug(t *testing.T) {
	ctx := context.Background()
	logger := testr.NewWithOptions(t, testr.Options{LogTimestamp: true, Verbosity: 10})
	ctx = logr.NewContext(ctx, logger)

	os.Setenv("KUBEBUILDER_ASSETS", projectRoot)

	kubeClients := getKubeClients(t, ctx)

	namespace := "test-namespace"
	ctrl.SetLogger(logger)
	klog.SetLogger(logger)

	{
		_, err := envtest.InstallCRDs(kubeClients.Rest, envtest.CRDInstallOptions{
			Paths: []string{
				path.Join(projectRoot, "example.com_mycrds.yaml"),
			},
			ErrorIfPathMissing: true,
		})
		require.NoError(t, err)
	}

	{
		ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
		err := kubeClients.Client.Create(ctx, &ns)
		require.NoError(t, err)
	}

	for i := 0; i < 400; i++ {
		testName := "test-" + strconv.Itoa(i)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			restClient := kubeClients.KubeClient.RESTClient()

			body := "{\"kind\":\"MyCRD\",\"apiVersion\":\"example.com/v1alpha1\",\"metadata\":{\"name\":\"" + testName + "\",\"namespace\":\"" + namespace + "\",\"creationTimestamp\":null},\"spec\":{\"prop1\":{\"prop1a\":\"testval\"}}}\n"

			err := restClient.Post().
				AbsPath("/apis/example.com/v1alpha1/namespaces/"+namespace+"/mycrds/").
				SetHeader("Content-Type", runtime.ContentTypeJSON).
				Body([]byte(body)).
				Do(ctx).
				Error()

			require.NoError(t, err)
		})
	}
}

type KubeClients struct {
	EnvTest    *envtest.Environment
	Scheme     *runtime.Scheme
	Rest       *rest.Config
	KubeClient *kubernetes.Clientset
	Client     client.WithWatch
}

func getKubeClients(tb testing.TB, ctx context.Context) *KubeClients {
	envTest := &envtest.Environment{}

	restConfig, err := envTest.Start()
	require.NoError(tb, err)

	tb.Cleanup(func() {
		tb.Log("Waiting for testEnv to exit")
		require.NoError(tb, envTest.Stop())
	})

	newScheme := runtime.NewScheme()
	controllerClient, err := client.NewWithWatch(restConfig, client.Options{Scheme: newScheme})
	require.NoError(tb, err)

	kubeClientset, err := kubernetes.NewForConfig(restConfig)
	require.NoError(tb, err)

	require.NoError(tb, corev1.AddToScheme(newScheme))
	require.NoError(tb, rbacv1.AddToScheme(newScheme))

	return &KubeClients{
		Scheme:     newScheme,
		Rest:       restConfig,
		KubeClient: kubeClientset,
		Client:     controllerClient,
	}
}
