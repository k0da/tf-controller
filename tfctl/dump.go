package tfctl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	json2 "encoding/json"
	"github.com/fluxcd/pkg/untar"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-exec/tfexec"
	infrav1 "github.com/weaveworks/tf-controller/api/v1alpha2"
	"github.com/weaveworks/tf-controller/utils"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *CLI) DumpTerraform(out io.Writer, resource string, dir string) error {
	ctx := context.TODO()
	httpClient := retryablehttp.NewClient()
	tfExec := os.Getenv("TF_BIN")
	if tfExec == "" {
		return errors.New("TF_BIN env var is unset")
	}

	key := types.NamespacedName{
		Name:      resource,
		Namespace: c.namespace,
	}

	fmt.Fprintf(out, " %s/%s Unpacking terraform resource into %s\n", c.namespace, resource, dir)
	terraform, err := getTerraform(context.TODO(), c.client, key)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}

	tfdir := filepath.Join(dir, terraform.Spec.Path)
	sourceObj, err := utils.GetSource(ctx, c.client, terraform)
	terraformData, err := utils.DownloadAsBytes(*sourceObj.GetArtifact(), httpClient)
	if err != nil {
		return err
	}
	_, err = untar.Untar(terraformData, dir)
	if err != nil {
		return err
	}
	tf, err := tfexec.NewTerraform(tfdir, tfExec)
	if err != nil {
		return err
	}
	tf.SetStdout(out)
	fmt.Fprintf(out, " Configuring backend \n")
	backendOpts, err := getBackendOpts(ctx, c.client, terraform, dir)
	if err != nil {
		return err
	}
	err = tf.Init(ctx, backendOpts...)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, " Selecting workspace: %s\n", terraform.Spec.Workspace)
	err = tf.WorkspaceSelect(ctx, terraform.Spec.Workspace)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, " Generating vars\n")
	err = generateVars(ctx, c.client, terraform, tfdir)
	if err != nil {
		return err
	}
	return nil
}

func generateVars(ctx context.Context, client client.Client, terraform *infrav1.Terraform, dir string) error {
	vars := map[string]*apiextensionsv1.JSON{}
	if terraform.Spec.Vars != nil && len(terraform.Spec.Vars) > 0 {
		for _, v := range terraform.Spec.Vars {
			vars[v.Name] = v.Value
		}
	}
	for _, vf := range terraform.Spec.VarsFrom {
		objectKey := types.NamespacedName{
			Namespace: terraform.Namespace,
			Name:      vf.Name,
		}
		data, err := utils.GetSecretConfigMapData(ctx, client, objectKey, vf.Kind)
		if err != nil && vf.Optional == false {
			return err
		}
		// if VarsKeys is null, use all
		if vf.VarsKeys == nil {
			for key, val := range data {
				vars[key], err = utils.JSONEncodeBytes(val)
				if err != nil {
					err := fmt.Errorf("failed to encode key %s with error: %w", key, err)
					return err
				}
			}
		} else {
			for _, pattern := range vf.VarsKeys {
				oldKey, newKey, err := utils.ParseRenamePattern(pattern)
				if err != nil {
					return err
				}

				vars[newKey], err = utils.JSONEncodeBytes(data[oldKey])
				if err != nil {
					err := fmt.Errorf("failed to encode key %q with error: %w", pattern, err)
					return err
				}
			}
		}
	}

	jsonBytes, err := json2.Marshal(vars)
	if err != nil {
		return err
	}

	varFilePath := filepath.Join(dir, "generated.auto.tfvars.json")
	if err := os.WriteFile(varFilePath, jsonBytes, 0644); err != nil {
		err = fmt.Errorf("error generating var file: %s", err)
		return err
	}
	return nil
}

func getBackendOpts(ctx context.Context, client client.Client, terraform *infrav1.Terraform, dir string) ([]tfexec.InitOption, error) {
	backendConfigsOpts := []tfexec.InitOption{}
	for _, bf := range terraform.Spec.BackendConfigsFrom {
		objectKey := types.NamespacedName{
			Namespace: terraform.Namespace,
			Name:      bf.Name,
		}

		backendData, err := utils.GetSecretConfigMapData(ctx, client, objectKey, bf.Kind)
		if err != nil && bf.Optional == false {
			return nil, err
		}
		// if VarsKeys is null, use all
		if bf.Keys == nil {
			for key, val := range backendData {
				backendConfigsOpts = append(backendConfigsOpts, tfexec.BackendConfig(key+"="+string(val)))
			}
		} else {
			for _, key := range bf.Keys {
				backendConfigsOpts = append(backendConfigsOpts, tfexec.BackendConfig(key+"="+string(backendData[key])))
			}
		}
	}
	initOpts := []tfexec.InitOption{tfexec.Upgrade(true)}
	initOpts = append(initOpts, backendConfigsOpts...)

	return initOpts, nil
}

func getTerraform(ctx context.Context, client client.Client, name types.NamespacedName) (*infrav1.Terraform, error) {
	terraform := &infrav1.Terraform{}
	if err := client.Get(ctx, name, terraform); err != nil {
		return nil, err
	}
	return terraform, nil
}
