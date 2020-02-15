/*
Copyright 2020 Talos Systems, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	metalv1alpha1 "github.com/talos-systems/metal-controller-manager/api/v1alpha1"
)

// EnvironmentReconciler reconciles a Environment object
type EnvironmentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=metal.arges.dev,resources=environments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metal.arges.dev,resources=environments/status,verbs=get;update;patch

func (r *EnvironmentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.reconcile(req)
}

func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&metalv1alpha1.Environment{}).
		Complete(r)
}
func (r *EnvironmentReconciler) reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	l := r.Log.WithValues("environment", req.Name)

	var env metalv1alpha1.Environment

	if err := r.Get(ctx, req.NamespacedName, &env); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("unable to get environment: %w", err)
	}

	envs := filepath.Join("/var/lib/arges/env", env.GetName())

	if _, err := os.Stat(envs); os.IsNotExist(err) {
		if err = os.MkdirAll(envs, 0777); err != nil {
			return ctrl.Result{}, fmt.Errorf("error creating asset directory: %w", err)
		}

		var (
			wg     sync.WaitGroup
			result *multierror.Error
		)

		wg.Add(2)

		for _, url := range []string{env.Spec.Kernel.URL, env.Spec.Initrd.URL} {
			go func(u string) {
				defer wg.Done()

				if u == "" {
					result = multierror.Append(result, errors.New("missing URL"))

					return
				}

				l.Info("saving asset", "url", u)

				if err = save(u, envs); err != nil {
					result = multierror.Append(result, fmt.Errorf("error saving %q: %w", u, err))
				}
			}(url)
		}

		wg.Wait()

		if result.ErrorOrNil() != nil {
			return ctrl.Result{}, result.ErrorOrNil()
		}
	}

	l.Info("all assets saved")

	return ctrl.Result{}, nil
}

func save(url, assets string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		asset := filepath.Base(url)

		f := filepath.Join(assets, asset)

		w, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}

		defer w.Close()

		r := resp.Body

		if _, err := io.Copy(w, r); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("failed to download asset: %d", resp.StatusCode)
	}

	return nil
}
