package management

import (
	"context"

	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/controllers/management/k3sUpgrade"
	"github.com/rancher/rancher/pkg/wrangler"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
)

func RegisterWrangler(ctx context.Context, wranglerContext *wrangler.Context, management *config.ManagementContext, manager *clustermanager.Manager) {
	// Add controllers to register here

	err := k3sUpgrade.Register(ctx, wranglerContext, management, manager)
	if err != nil {
		logrus.Fatal("Boom")
	}

}
