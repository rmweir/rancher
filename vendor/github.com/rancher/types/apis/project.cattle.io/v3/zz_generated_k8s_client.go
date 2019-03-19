package v3

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"github.com/rancher/norman/objectclient/dynamic"
	"github.com/rancher/norman/restwatch"
	"k8s.io/client-go/rest"
)

type (
	contextKeyType        struct{}
	contextClientsKeyType struct{}
)

type Interface interface {
	RESTClient() rest.Interface
	controller.Starter

	ServiceAccountTokensGetter
	DockerCredentialsGetter
	CertificatesGetter
	BasicAuthsGetter
	SSHAuthsGetter
	NamespacedServiceAccountTokensGetter
	NamespacedDockerCredentialsGetter
	NamespacedCertificatesGetter
	NamespacedBasicAuthsGetter
	NamespacedSSHAuthsGetter
	WorkloadsGetter
	AppsGetter
	AppRevisionsGetter
	SourceCodeProvidersGetter
	SourceCodeProviderConfigsGetter
	SourceCodeCredentialsGetter
	PipelinesGetter
	PipelineExecutionsGetter
	PipelineSettingsGetter
	SourceCodeRepositoriesGetter
}

type Clients struct {
	Interface Interface

	ServiceAccountToken           ServiceAccountTokenClient
	DockerCredential              DockerCredentialClient
	Certificate                   CertificateClient
	BasicAuth                     BasicAuthClient
	SSHAuth                       SSHAuthClient
	NamespacedServiceAccountToken NamespacedServiceAccountTokenClient
	NamespacedDockerCredential    NamespacedDockerCredentialClient
	NamespacedCertificate         NamespacedCertificateClient
	NamespacedBasicAuth           NamespacedBasicAuthClient
	NamespacedSSHAuth             NamespacedSSHAuthClient
	Workload                      WorkloadClient
	App                           AppClient
	AppRevision                   AppRevisionClient
	SourceCodeProvider            SourceCodeProviderClient
	SourceCodeProviderConfig      SourceCodeProviderConfigClient
	SourceCodeCredential          SourceCodeCredentialClient
	Pipeline                      PipelineClient
	PipelineExecution             PipelineExecutionClient
	PipelineSetting               PipelineSettingClient
	SourceCodeRepository          SourceCodeRepositoryClient
}

type Client struct {
	sync.Mutex
	restClient rest.Interface
	starters   []controller.Starter

	serviceAccountTokenControllers           map[string]ServiceAccountTokenController
	dockerCredentialControllers              map[string]DockerCredentialController
	certificateControllers                   map[string]CertificateController
	basicAuthControllers                     map[string]BasicAuthController
	sshAuthControllers                       map[string]SSHAuthController
	namespacedServiceAccountTokenControllers map[string]NamespacedServiceAccountTokenController
	namespacedDockerCredentialControllers    map[string]NamespacedDockerCredentialController
	namespacedCertificateControllers         map[string]NamespacedCertificateController
	namespacedBasicAuthControllers           map[string]NamespacedBasicAuthController
	namespacedSshAuthControllers             map[string]NamespacedSSHAuthController
	workloadControllers                      map[string]WorkloadController
	appControllers                           map[string]AppController
	appRevisionControllers                   map[string]AppRevisionController
	sourceCodeProviderControllers            map[string]SourceCodeProviderController
	sourceCodeProviderConfigControllers      map[string]SourceCodeProviderConfigController
	sourceCodeCredentialControllers          map[string]SourceCodeCredentialController
	pipelineControllers                      map[string]PipelineController
	pipelineExecutionControllers             map[string]PipelineExecutionController
	pipelineSettingControllers               map[string]PipelineSettingController
	sourceCodeRepositoryControllers          map[string]SourceCodeRepositoryController
}

func Factory(ctx context.Context, config rest.Config) (context.Context, controller.Starter, error) {
	c, err := NewForConfig(config)
	if err != nil {
		return ctx, nil, err
	}

	cs := NewClientsFromInterface(c)

	ctx = context.WithValue(ctx, contextKeyType{}, c)
	ctx = context.WithValue(ctx, contextClientsKeyType{}, cs)
	return ctx, c, nil
}

func ClientsFrom(ctx context.Context) *Clients {
	return ctx.Value(contextClientsKeyType{}).(*Clients)
}

func From(ctx context.Context) Interface {
	return ctx.Value(contextKeyType{}).(Interface)
}

func NewClients(config rest.Config) (*Clients, error) {
	iface, err := NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return NewClientsFromInterface(iface), nil
}

func NewClientsFromInterface(iface Interface) *Clients {
	return &Clients{
		Interface: iface,

		ServiceAccountToken: &serviceAccountTokenClient2{
			iface: iface.ServiceAccountTokens(""),
		},
		DockerCredential: &dockerCredentialClient2{
			iface: iface.DockerCredentials(""),
		},
		Certificate: &certificateClient2{
			iface: iface.Certificates(""),
		},
		BasicAuth: &basicAuthClient2{
			iface: iface.BasicAuths(""),
		},
		SSHAuth: &sshAuthClient2{
			iface: iface.SSHAuths(""),
		},
		NamespacedServiceAccountToken: &namespacedServiceAccountTokenClient2{
			iface: iface.NamespacedServiceAccountTokens(""),
		},
		NamespacedDockerCredential: &namespacedDockerCredentialClient2{
			iface: iface.NamespacedDockerCredentials(""),
		},
		NamespacedCertificate: &namespacedCertificateClient2{
			iface: iface.NamespacedCertificates(""),
		},
		NamespacedBasicAuth: &namespacedBasicAuthClient2{
			iface: iface.NamespacedBasicAuths(""),
		},
		NamespacedSSHAuth: &namespacedSshAuthClient2{
			iface: iface.NamespacedSSHAuths(""),
		},
		Workload: &workloadClient2{
			iface: iface.Workloads(""),
		},
		App: &appClient2{
			iface: iface.Apps(""),
		},
		AppRevision: &appRevisionClient2{
			iface: iface.AppRevisions(""),
		},
		SourceCodeProvider: &sourceCodeProviderClient2{
			iface: iface.SourceCodeProviders(""),
		},
		SourceCodeProviderConfig: &sourceCodeProviderConfigClient2{
			iface: iface.SourceCodeProviderConfigs(""),
		},
		SourceCodeCredential: &sourceCodeCredentialClient2{
			iface: iface.SourceCodeCredentials(""),
		},
		Pipeline: &pipelineClient2{
			iface: iface.Pipelines(""),
		},
		PipelineExecution: &pipelineExecutionClient2{
			iface: iface.PipelineExecutions(""),
		},
		PipelineSetting: &pipelineSettingClient2{
			iface: iface.PipelineSettings(""),
		},
		SourceCodeRepository: &sourceCodeRepositoryClient2{
			iface: iface.SourceCodeRepositories(""),
		},
	}
}

func NewForConfig(config rest.Config) (Interface, error) {
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = dynamic.NegotiatedSerializer
	}

	restClient, err := restwatch.UnversionedRESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &Client{
		restClient: restClient,

		serviceAccountTokenControllers:           map[string]ServiceAccountTokenController{},
		dockerCredentialControllers:              map[string]DockerCredentialController{},
		certificateControllers:                   map[string]CertificateController{},
		basicAuthControllers:                     map[string]BasicAuthController{},
		sshAuthControllers:                       map[string]SSHAuthController{},
		namespacedServiceAccountTokenControllers: map[string]NamespacedServiceAccountTokenController{},
		namespacedDockerCredentialControllers:    map[string]NamespacedDockerCredentialController{},
		namespacedCertificateControllers:         map[string]NamespacedCertificateController{},
		namespacedBasicAuthControllers:           map[string]NamespacedBasicAuthController{},
		namespacedSshAuthControllers:             map[string]NamespacedSSHAuthController{},
		workloadControllers:                      map[string]WorkloadController{},
		appControllers:                           map[string]AppController{},
		appRevisionControllers:                   map[string]AppRevisionController{},
		sourceCodeProviderControllers:            map[string]SourceCodeProviderController{},
		sourceCodeProviderConfigControllers:      map[string]SourceCodeProviderConfigController{},
		sourceCodeCredentialControllers:          map[string]SourceCodeCredentialController{},
		pipelineControllers:                      map[string]PipelineController{},
		pipelineExecutionControllers:             map[string]PipelineExecutionController{},
		pipelineSettingControllers:               map[string]PipelineSettingController{},
		sourceCodeRepositoryControllers:          map[string]SourceCodeRepositoryController{},
	}, nil
}

func (c *Client) RESTClient() rest.Interface {
	return c.restClient
}

func (c *Client) Sync(ctx context.Context) error {
	return controller.Sync(ctx, c.starters...)
}

func (c *Client) Start(ctx context.Context, threadiness int) error {
	return controller.Start(ctx, threadiness, c.starters...)
}

type ServiceAccountTokensGetter interface {
	ServiceAccountTokens(namespace string) ServiceAccountTokenInterface
}

func (c *Client) ServiceAccountTokens(namespace string) ServiceAccountTokenInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &ServiceAccountTokenResource, ServiceAccountTokenGroupVersionKind, serviceAccountTokenFactory{})
	return &serviceAccountTokenClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type DockerCredentialsGetter interface {
	DockerCredentials(namespace string) DockerCredentialInterface
}

func (c *Client) DockerCredentials(namespace string) DockerCredentialInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &DockerCredentialResource, DockerCredentialGroupVersionKind, dockerCredentialFactory{})
	return &dockerCredentialClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type CertificatesGetter interface {
	Certificates(namespace string) CertificateInterface
}

func (c *Client) Certificates(namespace string) CertificateInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &CertificateResource, CertificateGroupVersionKind, certificateFactory{})
	return &certificateClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type BasicAuthsGetter interface {
	BasicAuths(namespace string) BasicAuthInterface
}

func (c *Client) BasicAuths(namespace string) BasicAuthInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &BasicAuthResource, BasicAuthGroupVersionKind, basicAuthFactory{})
	return &basicAuthClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type SSHAuthsGetter interface {
	SSHAuths(namespace string) SSHAuthInterface
}

func (c *Client) SSHAuths(namespace string) SSHAuthInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &SSHAuthResource, SSHAuthGroupVersionKind, sshAuthFactory{})
	return &sshAuthClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type NamespacedServiceAccountTokensGetter interface {
	NamespacedServiceAccountTokens(namespace string) NamespacedServiceAccountTokenInterface
}

func (c *Client) NamespacedServiceAccountTokens(namespace string) NamespacedServiceAccountTokenInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &NamespacedServiceAccountTokenResource, NamespacedServiceAccountTokenGroupVersionKind, namespacedServiceAccountTokenFactory{})
	return &namespacedServiceAccountTokenClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type NamespacedDockerCredentialsGetter interface {
	NamespacedDockerCredentials(namespace string) NamespacedDockerCredentialInterface
}

func (c *Client) NamespacedDockerCredentials(namespace string) NamespacedDockerCredentialInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &NamespacedDockerCredentialResource, NamespacedDockerCredentialGroupVersionKind, namespacedDockerCredentialFactory{})
	return &namespacedDockerCredentialClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type NamespacedCertificatesGetter interface {
	NamespacedCertificates(namespace string) NamespacedCertificateInterface
}

func (c *Client) NamespacedCertificates(namespace string) NamespacedCertificateInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &NamespacedCertificateResource, NamespacedCertificateGroupVersionKind, namespacedCertificateFactory{})
	return &namespacedCertificateClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type NamespacedBasicAuthsGetter interface {
	NamespacedBasicAuths(namespace string) NamespacedBasicAuthInterface
}

func (c *Client) NamespacedBasicAuths(namespace string) NamespacedBasicAuthInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &NamespacedBasicAuthResource, NamespacedBasicAuthGroupVersionKind, namespacedBasicAuthFactory{})
	return &namespacedBasicAuthClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type NamespacedSSHAuthsGetter interface {
	NamespacedSSHAuths(namespace string) NamespacedSSHAuthInterface
}

func (c *Client) NamespacedSSHAuths(namespace string) NamespacedSSHAuthInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &NamespacedSSHAuthResource, NamespacedSSHAuthGroupVersionKind, namespacedSshAuthFactory{})
	return &namespacedSshAuthClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type WorkloadsGetter interface {
	Workloads(namespace string) WorkloadInterface
}

func (c *Client) Workloads(namespace string) WorkloadInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &WorkloadResource, WorkloadGroupVersionKind, workloadFactory{})
	logrus.Info("TEST enter workloads")
	return &workloadClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type AppsGetter interface {
	Apps(namespace string) AppInterface
}

func (c *Client) Apps(namespace string) AppInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &AppResource, AppGroupVersionKind, appFactory{})
	return &appClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type AppRevisionsGetter interface {
	AppRevisions(namespace string) AppRevisionInterface
}

func (c *Client) AppRevisions(namespace string) AppRevisionInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &AppRevisionResource, AppRevisionGroupVersionKind, appRevisionFactory{})
	return &appRevisionClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type SourceCodeProvidersGetter interface {
	SourceCodeProviders(namespace string) SourceCodeProviderInterface
}

func (c *Client) SourceCodeProviders(namespace string) SourceCodeProviderInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &SourceCodeProviderResource, SourceCodeProviderGroupVersionKind, sourceCodeProviderFactory{})
	return &sourceCodeProviderClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type SourceCodeProviderConfigsGetter interface {
	SourceCodeProviderConfigs(namespace string) SourceCodeProviderConfigInterface
}

func (c *Client) SourceCodeProviderConfigs(namespace string) SourceCodeProviderConfigInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &SourceCodeProviderConfigResource, SourceCodeProviderConfigGroupVersionKind, sourceCodeProviderConfigFactory{})
	return &sourceCodeProviderConfigClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type SourceCodeCredentialsGetter interface {
	SourceCodeCredentials(namespace string) SourceCodeCredentialInterface
}

func (c *Client) SourceCodeCredentials(namespace string) SourceCodeCredentialInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &SourceCodeCredentialResource, SourceCodeCredentialGroupVersionKind, sourceCodeCredentialFactory{})
	return &sourceCodeCredentialClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type PipelinesGetter interface {
	Pipelines(namespace string) PipelineInterface
}

func (c *Client) Pipelines(namespace string) PipelineInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &PipelineResource, PipelineGroupVersionKind, pipelineFactory{})
	return &pipelineClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type PipelineExecutionsGetter interface {
	PipelineExecutions(namespace string) PipelineExecutionInterface
}

func (c *Client) PipelineExecutions(namespace string) PipelineExecutionInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &PipelineExecutionResource, PipelineExecutionGroupVersionKind, pipelineExecutionFactory{})
	return &pipelineExecutionClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type PipelineSettingsGetter interface {
	PipelineSettings(namespace string) PipelineSettingInterface
}

func (c *Client) PipelineSettings(namespace string) PipelineSettingInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &PipelineSettingResource, PipelineSettingGroupVersionKind, pipelineSettingFactory{})
	return &pipelineSettingClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type SourceCodeRepositoriesGetter interface {
	SourceCodeRepositories(namespace string) SourceCodeRepositoryInterface
}

func (c *Client) SourceCodeRepositories(namespace string) SourceCodeRepositoryInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &SourceCodeRepositoryResource, SourceCodeRepositoryGroupVersionKind, sourceCodeRepositoryFactory{})
	return &sourceCodeRepositoryClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}
