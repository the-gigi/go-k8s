package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/the-gigi/go-k8s/pkg/client"
	"github.com/the-gigi/go-k8s/pkg/kind"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func main() {
	// Complete workflow example for go-k8s Kind package with go-k8s clients
	fmt.Println("Complete workflow example for go-k8s Kind package with go-k8s clients")

	// 1. Validate environment before starting
	fmt.Println("\n1. Validating environment...")
	if err := kind.ValidateEnvironment(); err != nil {
		log.Fatalf("Environment validation failed: %v", err)
	}
	fmt.Println("Environment validated successfully")

	// 2. Create KindClient with proper error handling
	fmt.Println("\n2. Creating Kind client...")
	_, err := kind.NewKindClient()
	if err != nil {
		log.Fatalf("Failed to create Kind client: %v", err)
	}
	fmt.Printf("Kind client created successfully\n")

	// 3. Create cluster with config file and context support
	fmt.Println("\n3. Creating cluster with context support...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	clusterName := "workflow-demo-cluster"
	options := kind.Options{
		Recreate:       true, // Delete if exists and recreate
		Wait:           2 * time.Minute, // Wait for cluster to be ready
		KubeConfigFile: "/tmp/kind-" + clusterName + "-kubeconfig", // Save kubeconfig to a file
		// ConfigFile: "path/to/kind-config.yaml", // Optional: custom config
	}

	cluster, err := kind.NewWithContext(ctx, clusterName, options)
	if err != nil {
		log.Fatalf("Failed to create cluster: %v", err)
	}
	fmt.Printf("Cluster '%s' created successfully with context support\n", clusterName)

	// Ensure cleanup
	defer func() {
		fmt.Println("\nCleaning up...")
		if err := cluster.DeleteWithContext(context.Background()); err != nil {
			log.Printf("Failed to delete cluster: %v", err)
		} else {
			fmt.Println("Cluster deleted successfully")
		}
	}()

	// 4. Set up go-k8s clients
	fmt.Println("\n4. Setting up go-k8s clients...")
	
	// Get kubeconfig path and context from the cluster
	kubeconfigPath := cluster.GetKubeConfig()
	if kubeconfigPath == "" {
		log.Fatalf("Failed to get kubeconfig path")
	}
	kubeContext := cluster.GetKubeContext()
	
	// Create go-k8s clientset
	clientset, err := client.NewClientset(kubeconfigPath, kubeContext)
	if err != nil {
		log.Fatalf("Failed to create go-k8s clientset: %v", err)
	}
	fmt.Println("Go-k8s clientset created successfully")
	
	// Create go-k8s dynamic client for more advanced operations
	dynamicClient, err := client.NewDynamicClient(kubeconfigPath, kubeContext)
	if err != nil {
		log.Fatalf("Failed to create go-k8s dynamic client: %v", err)
	}
	fmt.Println("Go-k8s dynamic client created successfully")

	// 5. Test context-aware operations
	fmt.Println("\n5. Testing context-aware operations...")
	nodeCtx, nodeCancel := context.WithTimeout(ctx, 30*time.Second)
	defer nodeCancel()

	nodes, err := cluster.GetNodesWithContext(nodeCtx)
	if err != nil {
		log.Printf("Failed to get nodes: %v", err)
	} else {
		fmt.Printf("Found %d nodes: %v\n", len(nodes), nodes)
	}

	// 6. Create and manage Kubernetes resources
	fmt.Println("\n6. Creating Kubernetes resources...")
	
	// Create a namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-app",
		},
	}
	
	_, err = clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create namespace: %v", err)
	} else {
		fmt.Println("Namespace 'demo-app' created successfully")
	}
	
	// Create a deployment
	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deployment",
			Namespace: "demo-app",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:alpine",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	
	_, err = clientset.AppsV1().Deployments("demo-app").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create deployment: %v", err)
	} else {
		fmt.Println("Deployment 'nginx-deployment' created successfully")
	}
	
	// Create a service to expose the deployment
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-service",
			Namespace: "demo-app",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "nginx",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	
	_, err = clientset.CoreV1().Services("demo-app").Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create service: %v", err)
	} else {
		fmt.Println("Service 'nginx-service' created successfully")
	}
	
	// 7. List and verify created resources
	fmt.Println("\n7. Listing and verifying created resources...")
	
	// List namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list namespaces: %v", err)
	} else {
		fmt.Printf("Total namespaces: %d\n", len(namespaces.Items))
		for _, ns := range namespaces.Items {
			if ns.Name == "demo-app" {
				fmt.Printf("✓ Found namespace: %s (Status: %s)\n", ns.Name, ns.Status.Phase)
			}
		}
	}
	
	// List deployments in demo-app namespace
	deployments, err := clientset.AppsV1().Deployments("demo-app").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list deployments: %v", err)
	} else {
		fmt.Printf("Deployments in demo-app namespace: %d\n", len(deployments.Items))
		for _, deploy := range deployments.Items {
			fmt.Printf("✓ Deployment: %s (Replicas: %d/%d)\n", 
				deploy.Name, deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
		}
	}
	
	// List services in demo-app namespace
	services, err := clientset.CoreV1().Services("demo-app").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list services: %v", err)
	} else {
		fmt.Printf("Services in demo-app namespace: %d\n", len(services.Items))
		for _, svc := range services.Items {
			fmt.Printf("✓ Service: %s (Type: %s, ClusterIP: %s)\n", 
				svc.Name, svc.Spec.Type, svc.Spec.ClusterIP)
		}
	}
	
	// List pods in demo-app namespace
	pods, err := clientset.CoreV1().Pods("demo-app").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list pods: %v", err)
	} else {
		fmt.Printf("Pods in demo-app namespace: %d\n", len(pods.Items))
		for _, pod := range pods.Items {
			fmt.Printf("✓ Pod: %s (Status: %s)\n", pod.Name, pod.Status.Phase)
		}
	}
	
	// Demonstrate dynamic client usage
	fmt.Println("\n7b. Demonstrating dynamic client capabilities...")
	
	// Use dynamic client to list deployments (alternative approach)
	deploymentGVR := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1", 
		Resource: "deployments",
	}
	
	unstructuredDeployments, err := dynamicClient.Resource(deploymentGVR).Namespace("demo-app").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list deployments with dynamic client: %v", err)
	} else {
		fmt.Printf("Dynamic client found %d deployments in demo-app namespace\n", len(unstructuredDeployments.Items))
		for _, item := range unstructuredDeployments.Items {
			fmt.Printf("✓ Dynamic client deployment: %s\n", item.GetName())
		}
	}

	// 8. Load Docker images
	fmt.Println("\n8. Loading Docker images...")
	imageCtx, imageCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer imageCancel()

	// Load a single image
	testImage := "hello-world:latest"
	fmt.Printf("Loading image: %s\n", testImage)
	if err := cluster.LoadDockerImageWithContext(imageCtx, testImage); err != nil {
		log.Printf("Failed to load image %s: %v", testImage, err)
	} else {
		fmt.Printf("Image %s loaded successfully\n", testImage)
	}

	// Load multiple images
	multipleImages := []string{"alpine:latest", "nginx:alpine"}
	fmt.Printf("Loading multiple images: %v\n", multipleImages)
	if err := cluster.LoadDockerImagesWithContext(imageCtx, multipleImages); err != nil {
		log.Printf("Failed to load multiple images: %v", err)
	} else {
		fmt.Printf("Multiple images loaded successfully\n")
	}

	// 9. Demonstrate error handling
	fmt.Println("\n9. Demonstrating error handling...")
	
	// Try to create a cluster with invalid options
	invalidCtx, invalidCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer invalidCancel()

	_, err = kind.NewWithContext(invalidCtx, "timeout-test", kind.Options{})
	if err != nil {
		fmt.Printf("Context timeout handled gracefully: %v\n", err)
	}

	// Try to load non-existent image
	if err := cluster.LoadDockerImageWithContext(ctx, "non-existent-image:latest"); err != nil {
		fmt.Printf("Image loading error handled gracefully: %v\n", err)
	}

	fmt.Println("\nComplete workflow finished successfully!")
	fmt.Println("\nFeatures demonstrated:")
	fmt.Println("  - Kind cluster creation and management")
	fmt.Println("  - Go-k8s client abstractions (clientset & dynamic client)")
	fmt.Println("  - Namespace creation and management")
	fmt.Println("  - Deployment creation with replica sets")
	fmt.Println("  - Service creation and exposure")
	fmt.Println("  - Resource listing and verification")
	fmt.Println("  - Context support for cancellation and timeouts")
	fmt.Println("  - Docker image loading functionality")
	fmt.Println("  - Proper error handling")
	fmt.Println("  - Environment validation and version checking")
}

// KindClient extension methods for the example
type KindClientExt struct {
	*kind.KindClient
}

func (c *KindClientExt) GetKindVersion() string {
	// This would access the kindVersion field if it were exported
	// For demo purposes, we'll return a placeholder
	return "v0.20.0+"
}