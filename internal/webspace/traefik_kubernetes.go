package webspace

import (
	"context"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	k8sCore "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8sMeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	k8sTypedCore "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	traefikConf "github.com/traefik/traefik/v2/pkg/config/dynamic"
	traefikClientset "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/generated/clientset/versioned"
	traefikTyped "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/generated/clientset/versioned/typed/traefik/v1alpha1"
	traefikCRD "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	traefikTypes "github.com/traefik/traefik/v2/pkg/types"

	cmCRD "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmMeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmClientset "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	cmTyped "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"

	"github.com/netsoc/webspaced/internal/config"
)

var k8sLabels = map[string]string{
	"app.kubernetes.io/managed-by": "webspaced",
}

// TraefikKubernetes manages webspace configuration for Traefik via Kubernetes resources
type TraefikKubernetes struct {
	config *config.Config

	epAPI  k8sTypedCore.EndpointsInterface
	svcAPI k8sTypedCore.ServiceInterface

	mwAPI    traefikTyped.MiddlewareInterface
	tcpMWAPI traefikTyped.MiddlewareTCPInterface
	irAPI    traefikTyped.IngressRouteInterface
	irTCPAPI traefikTyped.IngressRouteTCPInterface

	certManagerAPI cmTyped.CertificateInterface
}

// NewTraefikKubernetes manages webspace configuration for Traefik via Kubernetes resources
func NewTraefikKubernetes(cfg *config.Config) (Traefik, error) {
	k8sConf, err := clientcmd.BuildConfigFromFlags("", os.Getenv(clientcmd.RecommendedConfigPathEnvVar))
	if err != nil {
		return nil, fmt.Errorf("failed to load Kubernetes config: %w", err)
	}

	k8s, err := kubernetes.NewForConfig(k8sConf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	traefikK8s, err := traefikClientset.NewForConfig(k8sConf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Traefik Kubernetes client: %w", err)
	}

	cmK8s, err := cmClientset.NewForConfig(k8sConf)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert-manager Kubernetes client: %w", err)
	}

	return &TraefikKubernetes{
		config: cfg,

		epAPI:  k8s.CoreV1().Endpoints(cfg.Traefik.Kubernetes.Namespace),
		svcAPI: k8s.CoreV1().Services(cfg.Traefik.Kubernetes.Namespace),

		mwAPI:    traefikK8s.TraefikV1alpha1().Middlewares(cfg.Traefik.Kubernetes.Namespace),
		tcpMWAPI: traefikK8s.TraefikV1alpha1().MiddlewareTCPs(cfg.Traefik.Kubernetes.Namespace),
		irAPI:    traefikK8s.TraefikV1alpha1().IngressRoutes(cfg.Traefik.Kubernetes.Namespace),
		irTCPAPI: traefikK8s.TraefikV1alpha1().IngressRouteTCPs(cfg.Traefik.Kubernetes.Namespace),

		certManagerAPI: cmK8s.CertmanagerV1().Certificates(cfg.Traefik.Kubernetes.Namespace),
	}, nil
}

func (t *TraefikKubernetes) ClearAll(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	selector := k8sMeta.FormatLabelSelector(&k8sMeta.LabelSelector{MatchLabels: k8sLabels})
	listOpts := k8sMeta.ListOptions{
		LabelSelector: selector,
	}

	if err := t.irTCPAPI.DeleteCollection(ctx, k8sMeta.DeleteOptions{}, listOpts); err != nil {
		return fmt.Errorf("failed to delete Traefik IngressRouteTCP CRDs: %w", err)
	}
	if err := t.irAPI.DeleteCollection(ctx, k8sMeta.DeleteOptions{}, listOpts); err != nil {
		return fmt.Errorf("failed to delete Traefik IngressRoute CRDs: %w", err)
	}

	if err := t.tcpMWAPI.DeleteCollection(ctx, k8sMeta.DeleteOptions{}, listOpts); err != nil {
		return fmt.Errorf("failed to delete Traefik TCP Middleware CRDs: %w", err)
	}
	if err := t.mwAPI.DeleteCollection(ctx, k8sMeta.DeleteOptions{}, listOpts); err != nil {
		return fmt.Errorf("failed to delete Traefik Middleware CRDs: %w", err)
	}

	if err := t.certManagerAPI.DeleteCollection(ctx, k8sMeta.DeleteOptions{}, listOpts); err != nil {
		return fmt.Errorf("failed to delete cert-manager Certificates: %w", err)
	}

	svcList, err := t.svcAPI.List(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list Services: %w", err)
	}

	errChan := make(chan error, len(svcList.Items))
	for _, s := range svcList.Items {
		n := s.Name
		go func() {
			if err := t.svcAPI.Delete(ctx, n, k8sMeta.DeleteOptions{}); err != nil {
				errChan <- fmt.Errorf("failed to delete Service %v: %w", n, err)
			}

			errChan <- nil
		}()
	}
	for i := 0; i < len(svcList.Items); i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}

	if err := t.epAPI.DeleteCollection(ctx, k8sMeta.DeleteOptions{}, listOpts); err != nil {
		return fmt.Errorf("failed to delete Endpoints: %w", err)
	}

	return nil
}

// ClearConfig cleans out any configuration for an instance
func (t *TraefikKubernetes) ClearConfig(ctx context.Context, n string) error {
	if _, err := t.irTCPAPI.Get(ctx, n, k8sMeta.GetOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Traefik IngressRouteTCP CRD: %w", err)
		}
	} else if err := t.irTCPAPI.Delete(ctx, n, k8sMeta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Traefik IngressRouteTCP CRD: %w", err)
	}
	if _, err := t.irAPI.Get(ctx, n, k8sMeta.GetOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Traefik IngressRoute CRD: %w", err)
		}
	} else if err := t.irAPI.Delete(ctx, n, k8sMeta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Traefik IngressRoute CRD: %w", err)
	}

	if _, err := t.tcpMWAPI.Get(ctx, n+"-boot", k8sMeta.GetOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Traefik TCP Middleware CRD: %w", err)
		}
	} else if err := t.tcpMWAPI.Delete(ctx, n+"-boot", k8sMeta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Traefik TCP Middleware CRD: %w", err)
	}
	if _, err := t.mwAPI.Get(ctx, n+"-boot", k8sMeta.GetOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Traefik Middleware CRD: %w", err)
		}
	} else if err := t.mwAPI.Delete(ctx, n+"-boot", k8sMeta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Traefik Middleware CRD: %w", err)
	}

	if _, err := t.certManagerAPI.Get(ctx, "tls-"+n, k8sMeta.GetOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get cert-manager Certificate: %w", err)
		}
	} else if err := t.certManagerAPI.Delete(ctx, "tls-"+n, k8sMeta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete cert-manager Certificate: %w", err)
	}

	if _, err := t.svcAPI.Get(ctx, n, k8sMeta.GetOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Service: %w", err)
		}
	} else if err := t.svcAPI.Delete(ctx, n, k8sMeta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Service: %w", err)
	}

	if _, err := t.epAPI.Get(ctx, n, k8sMeta.GetOptions{}); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Endpoints: %w", err)
		}
	} else if err := t.epAPI.Delete(ctx, n, k8sMeta.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete Endpoints: %w", err)
	}

	return nil
}

// GenerateConfig generates new Traefik configuration for a webspace
func (t *TraefikKubernetes) GenerateConfig(ctx context.Context, ws *Webspace, addr string) error {
	if addr == "" && t.config.Traefik.WebspacedURL == "" {
		// Traefik hooks (only used when webspaces aren't running) are disabled
		return nil
	}

	n := ws.InstanceName()

	user, err := ws.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	domains, err := ws.GetDomains(ctx)
	if err != nil {
		return fmt.Errorf("failed to get webspace domains: %w", err)
	}

	wsb := traefikConf.WebspaceBoot{
		URL:      t.config.Traefik.WebspacedURL,
		IAMToken: t.config.Traefik.IAMToken,
		UserID:   ws.UserID,
	}

	ep := k8sCore.Endpoints{
		ObjectMeta: k8sMeta.ObjectMeta{
			Name:   n,
			Labels: k8sLabels,
		},
		Subsets: []k8sCore.EndpointSubset{
			{
				Addresses: []k8sCore.EndpointAddress{
					{
						// We need a dummy server to satisfy Traefik + Kubernetes
						IP: "1.1.1.1",
					},
				},
				Ports: []k8sCore.EndpointPort{
					{
						Name:     "http",
						Port:     int32(ws.Config.HTTPPort),
						Protocol: k8sCore.ProtocolTCP,
					},
				},
			},
		},
	}
	if addr != "" {
		ep.Subsets[0].Addresses[0].IP = addr
	}
	if _, err := t.epAPI.Create(ctx, &ep, k8sMeta.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create Kubernetes Endpoints: %w", err)
	}

	svc := k8sCore.Service{
		ObjectMeta: k8sMeta.ObjectMeta{
			Name:   n,
			Labels: k8sLabels,
		},
		Spec: k8sCore.ServiceSpec{
			ClusterIP: "None",
			Ports: []k8sCore.ServicePort{
				{
					Name:     "http",
					Port:     int32(ws.Config.HTTPPort),
					Protocol: k8sCore.ProtocolTCP,
				},
			},
		},
	}
	if _, err := t.svcAPI.Create(ctx, &svc, k8sMeta.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create Kubernetes Service: %w", err)
	}

	if !ws.Config.SNIPassthrough {
		var tls traefikCRD.TLS
		// ws.Domains only contains custom domains
		if len(ws.Domains) == 0 || t.config.Traefik.Kubernetes.ClusterIssuer == "" {
			if t.config.Traefik.Kubernetes.ClusterIssuer == "" {
				log.WithField("user", user.Username).Warn("No ClusterIssuer is configured, ignoring custom domains")
			}

			tls = traefikCRD.TLS{
				SecretName: t.config.Traefik.Kubernetes.DefaultSecret,
				Domains: []traefikTypes.Domain{
					{
						Main: "*." + t.config.Webspaces.Domain,
						SANs: t.config.Traefik.DefaultSANs,
					},
				},
			}
		} else if len(ws.Domains) > 0 {
			s := "tls-" + n

			crt := cmCRD.Certificate{
				ObjectMeta: k8sMeta.ObjectMeta{
					Name:   s,
					Labels: k8sLabels,
				},
				Spec: cmCRD.CertificateSpec{
					SecretName: s,
					// Should only need the custom domains in the certificate, Traefik will automagically use the
					// wildcard cert
					DNSNames: ws.Domains,
					IssuerRef: cmMeta.ObjectReference{
						Kind: "ClusterIssuer",
						Name: t.config.Traefik.Kubernetes.ClusterIssuer,
					},
				},
			}
			if _, err := t.certManagerAPI.Create(ctx, &crt, k8sMeta.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create cert-manager Certificate: %w", err)
			}

			tls = traefikCRD.TLS{
				SecretName: s,
				Domains: []traefikTypes.Domain{
					{
						Main: user.Username + "." + t.config.Webspaces.Domain,
						SANs: ws.Domains,
					},
				},
			}
		}

		rules := make([]string, len(domains))
		for i, d := range domains {
			rules[i] = fmt.Sprintf("Host(`%v`)", d)
		}
		rule := strings.Join(rules, " || ")

		ir := traefikCRD.IngressRoute{
			ObjectMeta: k8sMeta.ObjectMeta{
				Name:   n,
				Labels: k8sLabels,
			},
			Spec: traefikCRD.IngressRouteSpec{
				EntryPoints: []string{t.config.Traefik.HTTPSEntryPoint},
				Routes: []traefikCRD.Route{
					{
						Kind:  "Rule",
						Match: rule,
						Services: []traefikCRD.Service{
							{
								LoadBalancerSpec: traefikCRD.LoadBalancerSpec{
									Kind: "Service",
									Name: n,

									Port: intstr.FromInt(int(ws.Config.HTTPPort)),
								},
							},
						},
					},
				},
				TLS: &tls,
			},
		}

		if addr == "" {
			m := traefikCRD.Middleware{
				ObjectMeta: k8sMeta.ObjectMeta{
					Name:   n + "-boot",
					Labels: k8sLabels,
				},
				Spec: traefikCRD.MiddlewareSpec{
					WebspaceBoot: &wsb,
				},
			}

			if _, err := t.mwAPI.Create(ctx, &m, k8sMeta.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create WebspaceBoot middleware: %w", err)
			}

			ir.Spec.Routes[0].Middlewares = []traefikCRD.MiddlewareRef{
				{
					Name: m.ObjectMeta.Name,
				},
			}
		}

		if _, err := t.irAPI.Create(ctx, &ir, k8sMeta.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create IngressRoute CRD: %w", err)
		}
	} else {
		rules := make([]string, len(domains))
		for i, d := range domains {
			rules[i] = fmt.Sprintf("HostSNI(`%v`)", d)
		}
		rule := strings.Join(rules, " || ")

		ir := traefikCRD.IngressRouteTCP{
			ObjectMeta: k8sMeta.ObjectMeta{
				Name:   n,
				Labels: k8sLabels,
			},
			Spec: traefikCRD.IngressRouteTCPSpec{
				EntryPoints: []string{t.config.Traefik.HTTPSEntryPoint},
				Routes: []traefikCRD.RouteTCP{
					{
						Match: rule,
						Services: []traefikCRD.ServiceTCP{
							{
								Name: n,
								Port: intstr.FromInt(int(ws.Config.HTTPPort)),
							},
						},
					},
				},
				TLS: &traefikCRD.TLSTCP{
					Passthrough: true,
				},
			},
		}

		if addr == "" {
			m := traefikCRD.MiddlewareTCP{
				ObjectMeta: k8sMeta.ObjectMeta{
					Name:   n + "-boot",
					Labels: k8sLabels,
				},
				Spec: traefikCRD.MiddlewareTCPSpec{
					WebspaceBoot: &wsb,
				},
			}

			if _, err := t.tcpMWAPI.Create(ctx, &m, k8sMeta.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create WebspaceBoot TCP middleware: %w", err)
			}

			ir.Spec.Routes[0].Middlewares = []traefikCRD.ObjectReference{
				{
					Name: m.ObjectMeta.Name,
				},
			}
		}

		if _, err := t.irTCPAPI.Create(ctx, &ir, k8sMeta.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create IngressRoute CRD: %w", err)
		}
	}

	return nil
}
