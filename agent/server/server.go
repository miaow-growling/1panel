package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/cron"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/i18n"
	"github.com/1Panel-dev/1Panel/agent/init/app"
	"github.com/1Panel-dev/1Panel/agent/init/business"
	"github.com/1Panel-dev/1Panel/agent/init/cache"
	"github.com/1Panel-dev/1Panel/agent/init/db"
	"github.com/1Panel-dev/1Panel/agent/init/dir"
	"github.com/1Panel-dev/1Panel/agent/init/firewall"
	"github.com/1Panel-dev/1Panel/agent/init/hook"
	"github.com/1Panel-dev/1Panel/agent/init/lang"
	"github.com/1Panel-dev/1Panel/agent/init/log"
	"github.com/1Panel-dev/1Panel/agent/init/migration"
	"github.com/1Panel-dev/1Panel/agent/init/router"
	"github.com/1Panel-dev/1Panel/agent/init/validator"
	"github.com/1Panel-dev/1Panel/agent/init/viper"
	"github.com/1Panel-dev/1Panel/agent/utils/encrypt"
	"github.com/1Panel-dev/1Panel/agent/utils/files"
	"github.com/1Panel-dev/1Panel/agent/utils/re"
)

const (
	masterSocketDir          = "/etc/1panel"
	masterSocketPath         = masterSocketDir + "/agent.sock"
	masterSocketDirPerm      = 0o700
	masterSocketFilePerm     = 0o600
	masterSocketDirPermMask  = 0o077
	masterSocketFilePermMask = 0o077
)

func prepareMasterSocketDir(dir string) error {
	if err := os.MkdirAll(dir, masterSocketDirPerm); err != nil {
		return fmt.Errorf("create master socket dir %s failed: %w", dir, err)
	}
	if err := os.Chmod(dir, masterSocketDirPerm); err != nil {
		return fmt.Errorf("chmod master socket dir %s failed: %w", dir, err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat master socket dir %s failed: %w", dir, err)
	}
	if info.Mode().Perm()&masterSocketDirPermMask != 0 {
		return fmt.Errorf("master socket dir %s permission %#o is too permissive", dir, info.Mode().Perm())
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if int(stat.Uid) != os.Geteuid() {
			return fmt.Errorf(
				"master socket dir %s owner uid %d does not match current process uid %d",
				dir, stat.Uid, os.Geteuid(),
			)
		}
	}
	return nil
}

func secureMasterSocket(sockPath string) error {
	if err := os.Chmod(sockPath, masterSocketFilePerm); err != nil {
		return fmt.Errorf("chmod master socket %s failed: %w", sockPath, err)
	}
	info, err := os.Stat(sockPath)
	if err != nil {
		return fmt.Errorf("stat master socket %s failed: %w", sockPath, err)
	}
	if info.Mode().Perm()&masterSocketFilePermMask != 0 {
		return fmt.Errorf("master socket %s permission %#o is too permissive", sockPath, info.Mode().Perm())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if int(stat.Uid) != os.Geteuid() {
		return fmt.Errorf(
			"master socket %s owner uid %d does not match current process uid %d",
			sockPath, stat.Uid, os.Geteuid(),
		)
	}
	return nil
}

func Start() {
	re.Init()
	viper.Init()
	dir.Init()
	log.Init()
	global.LOG.Info("agent startup: logger initialized")
	if err := files.CleanupInterruptedDownloads(); err != nil {
		global.LOG.Warnf("clean interrupted remote downloads: %s", err)
	}
	db.Init()
	global.LOG.Info("agent startup: database initialized")
	migration.Init()
	global.LOG.Info("agent startup: migration initialized")
	i18n.Init()
	global.LOG.Info("agent startup: i18n initialized")
	cache.Init()
	global.LOG.Info("agent startup: cache initialized")
	app.Init()
	global.LOG.Info("agent startup: app initialized")
	lang.Init()
	global.LOG.Info("agent startup: language initialized")
	validator.Init()
	global.LOG.Info("agent startup: validator initialized")
	cron.Run()
	global.LOG.Info("agent startup: cron initialized")
	hook.Init()
	global.LOG.Info("agent startup: hook initialized")
	go firewall.Init()
	global.LOG.Info("agent startup: firewall init scheduled")
	InitOthers()
	global.LOG.Info("agent startup: edition initialized")

	rootRouter := router.Routers()
	global.LOG.Info("agent startup: router initialized")

	server := &http.Server{
		Handler: rootRouter,
	}

	if global.CONF.Base.Mode != "stable" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if global.IsMaster {
		global.LOG.Infof("agent startup: master mode, preparing unix socket %s", masterSocketPath)
		if err := prepareMasterSocketDir(masterSocketDir); err != nil {
			panic(err)
		}
		_ = os.Remove(masterSocketPath)
		listener, err := net.Listen("unix", masterSocketPath)
		if err != nil {
			panic(err)
		}
		if err := secureMasterSocket(masterSocketPath); err != nil {
			_ = listener.Close()
			panic(err)
		}
		global.LOG.Infof("agent startup: listening on unix socket %s", masterSocketPath)
		business.Init()
		global.LOG.Info("agent startup: business initialized")
		_ = server.Serve(listener)
		return
	} else {
		server.Addr = fmt.Sprintf("0.0.0.0:%s", global.CONF.Base.Port)
		global.LOG.Infof("agent startup: node mode, preparing https listener %s", server.Addr)
		settingRepo := repo.NewISettingRepo()
		var tlsCert tls.Certificate
		certItem, _ := settingRepo.GetValueByKey("ServerCrt")
		keyItem, _ := settingRepo.GetValueByKey("ServerKey")
		if certItem != "" && keyItem != "" {
			cert, _ := encrypt.StringDecrypt(certItem)
			key, _ := encrypt.StringDecrypt(keyItem)
			loaded, err := tls.X509KeyPair([]byte(cert), []byte(key))
			if err != nil {
				fmt.Printf("failed to load X.509 key pair: %s\n", err)
				return
			}
			tlsCert = loaded
		} else {
			// Community node mode: self-sign a server certificate when none is
			// provisioned; the master verifies via token signature, not PKI.
			generated, err := selfSignedCert()
			if err != nil {
				fmt.Printf("failed to generate self-signed certificate: %s\n", err)
				return
			}
			tlsCert = generated
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			ClientAuth:   tls.NoClientCert,
		}
		business.Init()
		global.LOG.Info("agent startup: business initialized")
		global.LOG.Infof("listen at https://0.0.0.0:%s", global.CONF.Base.Port)
		if err := server.ListenAndServeTLS("", ""); err != nil {
			panic(err)
		}
	}
}

// selfSignedCert generates an ephemeral TLS certificate for community node
// mode when no ServerCrt/ServerKey has been provisioned.
func selfSignedCert() (tls.Certificate, error) {
	host := ""
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				host = ipNet.IP.String()
				break
			}
		}
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "1panel-agent-node"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("0.0.0.0")},
		DNSNames:     []string{"localhost"},
	}
	if host != "" {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(host))
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPEM, keyPEM)
}

