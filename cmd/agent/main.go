package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/talos-systems/go-smbios/smbios"
	metal1alpha1 "github.com/talos-systems/metal-controller-manager/api/v1alpha1"
	"github.com/talos-systems/metal-controller-manager/pkg/client"
	"golang.org/x/sys/unix"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func print(s *smbios.Smbios) {
	println(s.BIOSInformation().Vendor())
	println(s.BIOSInformation().Version())
	println(s.BIOSInformation().ReleaseDate())
	println(s.SystemInformation().Manufacturer())
	println(s.SystemInformation().ProductName())
	println(s.SystemInformation().Version())
	println(s.SystemInformation().SerialNumber())
	println(s.SystemInformation().SKUNumber())
	println(s.SystemInformation().Family())
	println(s.BaseboardInformation().Manufacturer())
	println(s.BaseboardInformation().Product())
	println(s.BaseboardInformation().Version())
	println(s.BaseboardInformation().SerialNumber())
	println(s.BaseboardInformation().AssetTag())
	println(s.BaseboardInformation().LocationInChassis())
	println(s.SystemEnclosure().Manufacturer())
	println(s.SystemEnclosure().Version())
	println(s.SystemEnclosure().SerialNumber())
	println(s.SystemEnclosure().AssetTagNumber())
	println(s.SystemEnclosure().SKUNumber())
	println(s.ProcessorInformation().SocketDesignation())
	println(s.ProcessorInformation().ProcessorManufacturer())
	println(s.ProcessorInformation().ProcessorVersion())
	println(s.ProcessorInformation().SerialNumber())
	println(s.ProcessorInformation().AssetTag())
	println(s.ProcessorInformation().PartNumber())
	println(s.CacheInformation().SocketDesignation())
	println(s.PortConnectorInformation().InternalReferenceDesignator())
	println(s.PortConnectorInformation().ExternalReferenceDesignator())
	println(s.SystemSlots().SlotDesignation())
	println(s.BIOSLanguageInformation().CurrentLanguage())
	println(s.GroupAssociations().GroupName())
}

func main() {
	if err := os.MkdirAll("/dev", 0777); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("/proc", 0777); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("/sys", 0777); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("/tmp", 0777); err != nil {
		log.Fatal(err)
	}

	if err := unix.Mount("devtmpfs", "/dev", "devtmpfs", unix.MS_NOSUID, "mode=0755"); err != nil {
		log.Fatal(err)
	}

	if err := unix.Mount("proc", "/proc", "proc", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV, ""); err != nil {
		log.Fatal(err)
	}

	if err := unix.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
		log.Fatal(err)
	}

	if err := unix.Mount("tmpfs", "/tmp", "tmpfs", 0, ""); err != nil {
		log.Fatal(err)
	}

	s, err := smbios.New()
	if err != nil {
		log.Fatal(err)
	}

	print(s)

	resp, err := http.Get("http://192.168.1.10:8080/assets/arges/kubeconfig")
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("/tmp/kubeconfig", data, 0666); err != nil {
		log.Fatal(err)
	}

	var (
		config *rest.Config
	)

	config, err = clientcmd.BuildConfigFromFlags("", "/tmp/kubeconfig")
	if err != nil {
		log.Fatal(err)
	}

	c, err := client.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	uid, err := s.SystemInformation().UUID()
	if err != nil {
		log.Fatal(err)
	}

	obj := &metal1alpha1.Server{
		TypeMeta: v1.TypeMeta{
			Kind:       "Server",
			APIVersion: metal1alpha1.GroupVersion.Version,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      uid.String(),
			Namespace: "arges-system",
		},
	}

	_, err = controllerutil.CreateOrUpdate(context.Background(), c, obj, func() error {
		obj.Spec = metal1alpha1.ServerSpec{
			SystemInformation: &metal1alpha1.SystemInformation{
				Manufacturer: s.SystemInformation().Manufacturer(),
				ProductName:  s.SystemInformation().ProductName(),
				Version:      s.SystemInformation().Version(),
				SerialNumber: s.SystemInformation().SerialNumber(),
				SKUNumber:    s.SystemInformation().SKUNumber(),
				Family:       s.SystemInformation().Family(),
			},
			CPU: &metal1alpha1.CPUInformation{
				Manufacturer: s.ProcessorInformation().ProcessorManufacturer(),
				Version:      s.ProcessorInformation().ProcessorVersion(),
			},
		}

		return nil
	})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			log.Fatal(err)
		}

		log.Printf("%s already exists", uid.String())
	} else {
		log.Printf("Added %s", uid.String())
	}

	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
}
