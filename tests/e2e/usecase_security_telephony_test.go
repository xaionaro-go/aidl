//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genApp "github.com/AndroidGoLab/binder/android/app"
	genCredentials "github.com/AndroidGoLab/binder/android/credentials"
	genSecurity "github.com/AndroidGoLab/binder/android/security"
	genOemlock "github.com/AndroidGoLab/binder/android/service/oemlock"
	genKeystore2 "github.com/AndroidGoLab/binder/android/system/keystore2"
	"github.com/AndroidGoLab/binder/android/se/omapi"
	genOs "github.com/AndroidGoLab/binder/android/os"
	genTelephony "github.com/AndroidGoLab/binder/com/android/internal_/telephony"
	genEuicc "github.com/AndroidGoLab/binder/com/android/internal_/telephony/euicc"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// ==========================================================================
// Use case #46: Permission checker
// ==========================================================================

func TestUsecase_PermissionChecker(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "activity")

	am := genApp.NewActivityManagerProxy(svc)

	myPid := int32(os.Getpid())
	myUid := int32(os.Getuid())

	// Check INTERNET permission for our own process.
	result, err := am.CheckPermission(ctx, "android.permission.INTERNET", myPid, myUid)
	requireOrSkip(t, err)
	t.Logf("CheckPermission(INTERNET, pid=%d, uid=%d): %d", myPid, myUid, result)

	// Check INTERNET permission for root (uid=0) - should be granted.
	result, err = am.CheckPermission(ctx, "android.permission.INTERNET", myPid, 0)
	requireOrSkip(t, err)
	assert.Equal(t, int32(0), result, "root should have INTERNET permission")
	t.Logf("CheckPermission(INTERNET, pid=%d, uid=0): %d (0=GRANTED)", myPid, result)

	// Check CAMERA permission (likely denied for shell).
	result, err = am.CheckPermission(ctx, "android.permission.CAMERA", myPid, myUid)
	requireOrSkip(t, err)
	t.Logf("CheckPermission(CAMERA, pid=%d, uid=%d): %d", myPid, myUid, result)

	// Verify IsUserAMonkey (should be false in test).
	monkey, err := am.IsUserAMonkey(ctx)
	requireOrSkip(t, err)
	assert.False(t, monkey, "should not be a monkey in test")
	t.Logf("IsUserAMonkey: %v", monkey)

	// Process limit.
	limit, err := am.GetProcessLimit(ctx)
	requireOrSkip(t, err)
	t.Logf("GetProcessLimit: %d", limit)
}

// ==========================================================================
// Use case #47: Keystore operations (read-only)
// ==========================================================================

func TestUsecase_KeystoreOps(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	const svcName = "android.system.keystore2.IKeystoreService/default"
	svc, err := sm.CheckService(ctx, servicemanager.ServiceName(svcName))
	requireOrSkip(t, err)
	if svc == nil {
		t.Skip("keystore2 service not registered")
	}
	require.True(t, svc.IsAlive(ctx), "keystore2 should be alive")

	ks := genKeystore2.NewKeystoreServiceProxy(svc)

	// Query number of entries in SELINUX domain (namespace 0).
	count, err := ks.GetNumberOfEntries(ctx, genKeystore2.DomainSELINUX, 0)
	requireOrSkip(t, err)
	t.Logf("GetNumberOfEntries(SELINUX, ns=0): %d", count)
	require.GreaterOrEqual(t, count, int32(0))

	// Query number of entries in APP domain.
	appCount, err := ks.GetNumberOfEntries(ctx, genKeystore2.DomainAPP, -1)
	requireOrSkip(t, err)
	t.Logf("GetNumberOfEntries(APP, ns=-1): %d", appCount)
	require.GreaterOrEqual(t, appCount, int32(0))
}

// ==========================================================================
// Use case #48: Attestation verification service
// ==========================================================================

func TestUsecase_AttestationVerify(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	svc, err := sm.CheckService(ctx, servicemanager.AttestationVerificationService)
	requireOrSkip(t, err)
	if svc == nil {
		t.Skip("attestation_verification service not registered")
	}
	require.True(t, svc.IsAlive(ctx), "attestation_verification should be alive")
	t.Logf("attestation_verification: handle=%d, alive=true", svc.Handle())

	// Also check file_integrity service.
	fiSvc, err := sm.CheckService(ctx, servicemanager.FileIntegrityService)
	requireOrSkip(t, err)
	if fiSvc != nil {
		fiProxy := genSecurity.NewFileIntegrityServiceProxy(fiSvc)
		supported, err := fiProxy.IsApkVeritySupported(ctx)
		requireOrSkip(t, err)
		t.Logf("IsApkVeritySupported: %v", supported)
	} else {
		t.Log("file_integrity service not registered (optional)")
	}
}

// ==========================================================================
// Use case #49: OEM lock status
// ==========================================================================

func TestUsecase_OemLockStatus(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// Verify the service exists and is reachable. All OemLock query
	// methods require MANAGE_CARRIER_OEM_UNLOCK_STATE which shell
	// does not have, so we verify service liveness and try each
	// method independently to maximize coverage.
	svc, err := sm.CheckService(ctx, servicemanager.OemLockService)
	requireOrSkip(t, err)
	if svc == nil {
		t.Skip("oem_lock service not registered")
	}
	require.True(t, svc.IsAlive(ctx), "oem_lock service should be alive")
	t.Logf("oem_lock: handle=%d, alive=true", svc.Handle())

	proxy := genOemlock.NewOemLockServiceProxy(svc)

	// Try each method; they all require MANAGE_CARRIER_OEM_UNLOCK_STATE
	// or MANAGE_USER_OEM_UNLOCK_STATE, so permission denials are expected.
	lockName, err := proxy.GetLockName(ctx)
	if err != nil {
		t.Logf("GetLockName: %v (expected without carrier OEM unlock permission)", err)
	} else {
		t.Logf("Lock name: %q", lockName)
	}

	unlocked, err := proxy.IsDeviceOemUnlocked(ctx)
	if err != nil {
		t.Logf("IsDeviceOemUnlocked: %v (expected without permission)", err)
	} else {
		t.Logf("Device OEM unlocked: %v", unlocked)
	}
}

// ==========================================================================
// Use case #50: Secure element (OMAPI)
// ==========================================================================

func TestUsecase_SecureElement(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "secure_element")

	proxy := omapi.NewSecureElementServiceProxy(svc)

	readers, err := proxy.GetReaders(ctx)
	requireOrSkip(t, err)
	t.Logf("Secure element readers: %d", len(readers))
	for i, r := range readers {
		t.Logf("  [%d] %s", i, r)
	}
}

// ==========================================================================
// Use case #51: RKP monitor (remote key provisioning)
// ==========================================================================

func TestUsecase_RkpMonitor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// Check remote_provisioning service.
	svc, err := sm.CheckService(ctx, servicemanager.RemoteProvisioningService)
	requireOrSkip(t, err)
	if svc == nil {
		t.Skip("remote_provisioning service not registered")
	}
	require.True(t, svc.IsAlive(ctx), "remote_provisioning should be alive")
	t.Logf("remote_provisioning: handle=%d, alive=true", svc.Handle())

	// Check security_state service.
	secSvc, err := sm.CheckService(ctx, servicemanager.SecurityStateService)
	requireOrSkip(t, err)
	if secSvc != nil {
		secProxy := genOs.NewSecurityStateManagerProxy(secSvc)
		state, err := secProxy.GetGlobalSecurityState(ctx)
		requireOrSkip(t, err)
		t.Logf("GetGlobalSecurityState: %+v", state)
	} else {
		t.Log("security_state service not registered (optional)")
	}
}

// ==========================================================================
// Use case #52: Credential manager
// ==========================================================================

func TestUsecase_CredentialManager(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	proxy, err := genCredentials.GetCredentialManager(ctx, sm)
	requireOrSkip(t, err)

	enabled, err := proxy.IsServiceEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("Credential service enabled: %v", enabled)

	providers, err := proxy.GetCredentialProviderServices(ctx, 0)
	if err != nil {
		t.Logf("GetCredentialProviderServices returned error (may require permission): %v", err)
	} else {
		t.Logf("Credential providers: %d", len(providers))
	}
}

// ==========================================================================
// Use case #53: SIM status
// ==========================================================================

func TestUsecase_SimStatus(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "phone")

	phone := genTelephony.NewTelephonyProxy(svc)

	// The no-argument telephony methods (GetCallState, GetDataState, etc.)
	// are rejected on API > 30 with "This method can only be used for
	// applications targeting API version 30 or less." Use the
	// subscription/slot-based variants that pass callingPackage or subId.

	// Call state via subscription (0=IDLE, 1=RINGING, 2=OFFHOOK).
	callState, err := phone.GetCallStateForSubscription(ctx, 1, "")
	requireOrSkip(t, err)
	t.Logf("Call state (subId=1): %d (0=IDLE)", callState)
	require.GreaterOrEqual(t, callState, int32(0))
	require.LessOrEqual(t, callState, int32(2))

	// Data state via subscription.
	dataState, err := phone.GetDataStateForSubId(ctx, 1)
	requireOrSkip(t, err)
	t.Logf("Data state (subId=1): %d", dataState)

	// Data activity via subscription.
	dataActivity, err := phone.GetDataActivityForSubId(ctx, 1)
	requireOrSkip(t, err)
	t.Logf("Data activity (subId=1): %d", dataActivity)

	// Network country.
	country, err := phone.GetNetworkCountryIsoForPhone(ctx, 0)
	requireOrSkip(t, err)
	t.Logf("Network country (slot 0): %q", country)

	// Methods that may be restricted on API > 30 or removed on newer
	// API levels. Test independently so one skip doesn't block others.
	t.Run("HasIccCardUsingSlotIndex", func(t *testing.T) {
		hasIcc, err := phone.HasIccCardUsingSlotIndex(ctx, 0)
		requireOrSkip(t, err)
		t.Logf("ICC card present (slot 0): %v", hasIcc)
	})

	t.Run("IsRadioOn", func(t *testing.T) {
		radioOn, err := phone.IsRadioOn(ctx)
		requireOrSkip(t, err)
		t.Logf("Radio on: %v", radioOn)
	})
}

// ==========================================================================
// Use case #54: Dual SIM monitoring
// ==========================================================================

func TestUsecase_DualSim(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// Query subscription service.
	subSvc, err := sm.CheckService(ctx, servicemanager.ServiceName("isub"))
	requireOrSkip(t, err)
	if subSvc == nil {
		t.Skip("isub service not registered")
	}

	sub := genTelephony.NewSubProxy(subSvc)

	// Max SIM slots.
	maxSlots, err := sub.GetActiveSubInfoCountMax(ctx)
	requireOrSkip(t, err)
	t.Logf("Max SIM slots: %d", maxSlots)
	require.Greater(t, maxSlots, int32(0), "should have at least 1 SIM slot")

	// Active subscription count.
	activeCount, err := sub.GetActiveSubInfoCount(ctx, true)
	requireOrSkip(t, err)
	t.Logf("Active subscriptions: %d", activeCount)
	require.GreaterOrEqual(t, activeCount, int32(0))

	// Default subscription ID.
	defaultSubId, err := sub.GetDefaultSubId(ctx)
	requireOrSkip(t, err)
	t.Logf("Default subscription ID: %d", defaultSubId)

	// Active subscription IDs.
	activeSubIds, err := sub.GetActiveSubIdList(ctx, false)
	requireOrSkip(t, err)
	t.Logf("Active subscription IDs: %v", activeSubIds)

	// Per-slot ICC check via telephony.
	phoneSvc := getService(ctx, t, driver, "phone")
	phone := genTelephony.NewTelephonyProxy(phoneSvc)

	slotCount := maxSlots
	if slotCount > 4 {
		slotCount = 4 // cap to avoid excessive calls
	}
	for slot := int32(0); slot < slotCount; slot++ {
		hasIcc, err := phone.HasIccCardUsingSlotIndex(ctx, slot)
		if err != nil {
			t.Logf("  Slot %d HasIccCard: error: %v", slot, err)
			continue
		}
		t.Logf("  Slot %d ICC present: %v", slot, hasIcc)
	}

	// Multi-SIM support.
	multiSim, err := phone.IsMultiSimSupported(ctx)
	requireOrSkip(t, err)
	// 0=ALLOWED, 1=NOT_SUPPORTED_BY_HARDWARE, 2=NOT_SUPPORTED_BY_CARRIER
	t.Logf("IsMultiSimSupported: %d (0=ALLOWED)", multiSim)
}

// ==========================================================================
// Use case #55: eSIM/eUICC manager
// ==========================================================================

func TestUsecase_EsimManager(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// Get default eUICC card ID.
	phoneSvc := getService(ctx, t, driver, "phone")
	phone := genTelephony.NewTelephonyProxy(phoneSvc)

	cardId, err := phone.GetCardIdForDefaultEuicc(ctx, 0)
	if err != nil {
		t.Logf("GetCardIdForDefaultEuicc: %v (using cardId=0)", err)
		cardId = 0
	} else {
		t.Logf("Default eUICC card ID: %d", cardId)
	}

	// Check euicc service.
	euiccSvc, err := sm.CheckService(ctx, servicemanager.EuiccService)
	requireOrSkip(t, err)
	if euiccSvc == nil {
		t.Skip("euicc service not registered (device may not support eSIM)")
	}

	euiccProxy := genEuicc.NewEuiccControllerProxy(euiccSvc)

	// OTA status.
	otaStatus, err := euiccProxy.GetOtaStatus(ctx, cardId)
	requireOrSkip(t, err)
	t.Logf("eUICC OTA status: %d", otaStatus)

	// Supported countries.
	countries, err := euiccProxy.GetSupportedCountries(ctx, true)
	requireOrSkip(t, err)
	t.Logf("Supported eSIM countries: %d", len(countries))

	// eUICC info.
	info, err := euiccProxy.GetEuiccInfo(ctx, cardId)
	requireOrSkip(t, err)
	t.Logf("eUICC info: %+v", info)
}

// ==========================================================================
// Use case #56: SMS monitor
// ==========================================================================

func TestUsecase_SmsMonitor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "isms")

	sms := genTelephony.NewSmsProxy(svc)

	// Preferred SMS subscription.
	prefSub, err := sms.GetPreferredSmsSubscription(ctx)
	requireOrSkip(t, err)
	t.Logf("Preferred SMS subscription: %d", prefSub)

	// SMS prompt enabled (multi-SIM).
	promptEnabled, err := sms.IsSMSPromptEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("SMS prompt enabled: %v", promptEnabled)

	// IMS SMS support for default subscription.
	imsSupported, err := sms.IsImsSmsSupportedForSubscriber(ctx, 1)
	if err != nil {
		t.Logf("IsImsSmsSupportedForSubscriber(1): %v (may require permission)", err)
	} else {
		t.Logf("IMS SMS supported (subId=1): %v", imsSupported)
	}
}

// ==========================================================================
// Use case #57: Carrier configuration
// ==========================================================================

func TestUsecase_CarrierConfig(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "carrier_config")

	cc := genTelephony.NewCarrierConfigLoaderProxy(svc)

	// Default carrier service package.
	pkg, err := cc.GetDefaultCarrierServicePackageName(ctx)
	requireOrSkip(t, err)
	t.Logf("Default carrier service package: %q", pkg)

	// Carrier config for default subscription.
	config, err := cc.GetConfigForSubId(ctx, 1)
	if err != nil {
		t.Logf("GetConfigForSubId(1): %v (may require permission)", err)
	} else {
		t.Logf("Carrier config for subId=1: %+v", config)
	}
}

// ==========================================================================
// Use case #58: IMS registration monitor
// ==========================================================================

func TestUsecase_ImsMonitor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "phone")
	sm := servicemanager.New(driver)

	phone := genTelephony.NewTelephonyProxy(svc)

	// Check telephony_ims service availability first — this always works.
	imsSvc, err := sm.CheckService(ctx, servicemanager.TelephonyImsService)
	requireOrSkip(t, err)
	if imsSvc != nil {
		require.True(t, imsSvc.IsAlive(ctx), "telephony_ims should be alive")
		t.Logf("telephony_ims: handle=%d, alive=true", imsSvc.Handle())
	} else {
		t.Log("telephony_ims service not registered (optional)")
	}

	// IMS methods may fail with ServiceSpecific errors when no active
	// IMS subscription is available (common on devices without SIM or
	// with carriers that don't support IMS). Test each independently
	// so one failure doesn't block the others.
	t.Run("IsImsRegistered", func(t *testing.T) {
		registered, err := phone.IsImsRegistered(ctx, 1)
		requireOrSkip(t, err)
		t.Logf("IMS registered (subId=1): %v", registered)
	})

	t.Run("IsWifiCallingAvailable", func(t *testing.T) {
		wifiCalling, err := phone.IsWifiCallingAvailable(ctx, 1)
		requireOrSkip(t, err)
		t.Logf("WiFi calling available (subId=1): %v", wifiCalling)
	})

	t.Run("IsVideoTelephonyAvailable", func(t *testing.T) {
		videoAvail, err := phone.IsVideoTelephonyAvailable(ctx, 1)
		requireOrSkip(t, err)
		t.Logf("Video telephony available (subId=1): %v", videoAvail)
	})

	t.Run("IsVoWiFiSettingEnabled", func(t *testing.T) {
		vowifi, err := phone.IsVoWiFiSettingEnabled(ctx, 1)
		requireOrSkip(t, err)
		t.Logf("VoWiFi setting enabled (subId=1): %v", vowifi)
	})
}

// ==========================================================================
// Use case #59: Satellite telephony availability
// ==========================================================================

func TestUsecase_SatelliteCheck(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// Check satellite service.
	svc, err := sm.CheckService(ctx, servicemanager.SatelliteService)
	requireOrSkip(t, err)
	if svc == nil {
		t.Log("satellite service: NOT REGISTERED (device may not support satellite telephony)")
	} else {
		require.True(t, svc.IsAlive(ctx), "satellite service should be alive")
		t.Logf("satellite service: handle=%d, alive=true", svc.Handle())
	}

	// Check related services.
	services := []struct {
		name servicemanager.ServiceName
		desc string
	}{
		{servicemanager.TelephonyService, "phone"},
		{servicemanager.TelephonyImsService, "telephony_ims"},
		{servicemanager.TelephonyRegistryService, "telephony_registry"},
	}

	for _, s := range services {
		check, err := sm.CheckService(ctx, s.name)
		requireOrSkip(t, err)
		if check == nil {
			t.Logf("  %s: NOT REGISTERED", s.desc)
		} else {
			alive := check.IsAlive(ctx)
			t.Logf("  %s: handle=%d, alive=%v", s.desc, check.Handle(), alive)
		}
	}
}
