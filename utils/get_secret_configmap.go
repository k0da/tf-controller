package utils

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"maps"
)

func getSecret(ctx context.Context, c client.Client, name types.NamespacedName) (map[string][]byte, error) {
  obj := &corev1.Secret{}	
  err := c.Get(ctx, name, obj)
  if err != nil {
  	return nil, err
}
  return obj.Data, nil
}
func getConfigMap(ctx context.Context, c client.Client, name types.NamespacedName) (map[string][]byte, error) {
  obj := &corev1.ConfigMap{}	
  err := c.Get(ctx, name, obj)
  if err != nil {
  	return nil, err
  }
  data := make(map[string][]byte)
  if obj.Data != nil {
	for k, v := range obj.Data {
		data[k] = []byte(v)
	}
  }
  if obj.BinaryData != nil {
  	maps.Copy(data, obj.BinaryData)
  }

  return data, nil
}

func GetSecretConfigMapData(ctx context.Context,c client.Client, name types.NamespacedName, kind string) (map[string][]byte, error) {
	data := make(map[string][]byte)
	var err error
	if kind == "ConfigMap" {
		data, err = getConfigMap(ctx, c, name)
		if err != nil {
			return nil, err
		}
	}
	if kind == "Secret" {
		data, err = getSecret(ctx, c, name)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}