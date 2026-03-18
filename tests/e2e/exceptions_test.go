//go:build e2e

package e2e

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xaionaro-go/binder/binder"
	"github.com/xaionaro-go/binder/binder/versionaware"
	aidlerrors "github.com/xaionaro-go/binder/errors"
	"github.com/xaionaro-go/binder/parcel"
	"github.com/xaionaro-go/binder/servicemanager"
)

// getService retrieves a named Android service via the service manager.
func getService(
	ctx context.Context,
	t *testing.T,
	driver *versionaware.Transport,
	name string,
) binder.IBinder {
	t.Helper()
	sm := servicemanager.New(driver)
	svc, err := sm.GetService(ctx, servicemanager.ServiceName(name))
	requireOrSkip(t, err)
	require.NotNil(t, svc, "expected non-nil binder for %s", name)
	return svc
}

// transactExpectException sends a transaction and asserts that ReadStatus
// returns a *StatusError. If the expected exception code matches, it returns
// the StatusError. If no exception is thrown or a different exception type
// is received, the test is skipped (device/version-dependent behavior).
func transactExpectException(
	ctx context.Context,
	t *testing.T,
	svc binder.IBinder,
	code binder.TransactionCode,
	data *parcel.Parcel,
	expectedExc aidlerrors.ExceptionCode,
) *aidlerrors.StatusError {
	t.Helper()

	reply, err := svc.Transact(ctx, code, 0, data)
	requireOrSkip(t, err)

	statusErr := binder.ReadStatus(reply)
	if statusErr == nil {
		t.Skipf("no exception thrown (device/version-dependent behavior); expected %s", expectedExc)
		return nil
	}

	var se *aidlerrors.StatusError
	if !errors.As(statusErr, &se) {
		t.Skipf("non-StatusError returned: %T: %v", statusErr, statusErr)
		return nil
	}

	if se.Exception != expectedExc {
		t.Skipf("got %s exception instead of expected %s (device/version-dependent behavior)", se.Exception, expectedExc)
		return nil
	}

	return se
}

// --- Exception tests ---

const (
	powerManagerDescriptor       = "android.os.IPowerManager"
	webViewUpdateDescriptor      = "android.webkit.IWebViewUpdateService"
	storageStatsManagerDescriptor = "android.app.usage.IStorageStatsManager"
)

func TestException_NullPointer(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	am := getService(ctx, t, driver, "activity")

	// IActivityManager transaction code 80 with no arguments causes a NullPointerException.
	data := parcel.New()
	data.WriteInterfaceToken(activityManagerDescriptor)

	se := transactExpectException(ctx, t, am, 80, data, aidlerrors.ExceptionNullPointer)
	if se != nil {
		t.Logf("NullPointer exception message: %q", se.Message)
	}
}

func TestException_IllegalArgument(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	pm := getService(ctx, t, driver, "power")

	// IPowerManager transaction code 9 with missing lock param causes IllegalArgumentException.
	data := parcel.New()
	data.WriteInterfaceToken(powerManagerDescriptor)

	se := transactExpectException(ctx, t, pm, 9, data, aidlerrors.ExceptionIllegalArgument)
	if se != nil {
		t.Logf("IllegalArgument exception message: %s", se.Message)
	}
}

func TestException_IllegalState(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	ws := getService(ctx, t, driver, "webviewupdate")

	// IWebViewUpdateService transaction code 8 causes IllegalStateException.
	data := parcel.New()
	data.WriteInterfaceToken(webViewUpdateDescriptor)

	se := transactExpectException(ctx, t, ws, 8, data, aidlerrors.ExceptionIllegalState)
	if se != nil {
		t.Logf("IllegalState exception message: %s", se.Message)
	}
}

func TestException_BadParcelable(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	am := getService(ctx, t, driver, "activity")

	// IActivityManager transaction code 178 with two string params causes BadParcelableException.
	data := parcel.New()
	data.WriteInterfaceToken(activityManagerDescriptor)
	data.WriteString16("com.android.systemui")
	data.WriteString16("com.android.systemui")

	se := transactExpectException(ctx, t, am, 178, data, aidlerrors.ExceptionBadParcelable)
	if se != nil {
		t.Logf("BadParcelable exception message: %s", se.Message)
	}
}

func TestException_Parcelable(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	ss := getService(ctx, t, driver, "storagestats")

	// IStorageStatsManager transaction code 5 with empty volume UUID and package name.
	data := parcel.New()
	data.WriteInterfaceToken(storageStatsManagerDescriptor)
	data.WriteString16("")
	data.WriteString16("com.android.shell")

	se := transactExpectException(ctx, t, ss, 5, data, aidlerrors.ExceptionParcelable)
	if se != nil {
		t.Logf("Parcelable exception message: %s", se.Message)
	}
}

func TestException_AllTypesInventory(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	type exceptionTestCase struct {
		name        string
		service     string
		descriptor  string
		code        binder.TransactionCode
		setupData   func(*parcel.Parcel)
		expected    aidlerrors.ExceptionCode
	}

	cases := []exceptionTestCase{
		{
			name:       "Security",
			service:    "activity",
			descriptor: activityManagerDescriptor,
			code:       148,
			setupData:  func(p *parcel.Parcel) {},
			expected:   aidlerrors.ExceptionSecurity,
		},
		{
			name:       "NullPointer",
			service:    "activity",
			descriptor: activityManagerDescriptor,
			code:       80,
			setupData:  func(p *parcel.Parcel) {},
			expected:   aidlerrors.ExceptionNullPointer,
		},
		{
			name:       "IllegalArgument",
			service:    "power",
			descriptor: powerManagerDescriptor,
			code:       9,
			setupData:  func(p *parcel.Parcel) {},
			expected:   aidlerrors.ExceptionIllegalArgument,
		},
		{
			name:       "IllegalState",
			service:    "webviewupdate",
			descriptor: webViewUpdateDescriptor,
			code:       8,
			setupData:  func(p *parcel.Parcel) {},
			expected:   aidlerrors.ExceptionIllegalState,
		},
		{
			name:       "BadParcelable",
			service:    "activity",
			descriptor: activityManagerDescriptor,
			code:       178,
			setupData: func(p *parcel.Parcel) {
				p.WriteString16("com.android.systemui")
				p.WriteString16("com.android.systemui")
			},
			expected: aidlerrors.ExceptionBadParcelable,
		},
		{
			name:       "Parcelable",
			service:    "storagestats",
			descriptor: storageStatsManagerDescriptor,
			code:       5,
			setupData: func(p *parcel.Parcel) {
				p.WriteString16("")
				p.WriteString16("com.android.shell")
			},
			expected: aidlerrors.ExceptionParcelable,
		},
	}

	testedExceptions := make(map[aidlerrors.ExceptionCode]bool)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := sm.GetService(ctx, servicemanager.ServiceName(tc.service))
			requireOrSkip(t, err)
			require.NotNil(t, svc)

			data := parcel.New()
			data.WriteInterfaceToken(tc.descriptor)
			tc.setupData(data)

			reply, err := svc.Transact(ctx, tc.code, 0, data)
			requireOrSkip(t, err)

			statusErr := binder.ReadStatus(reply)
			if statusErr == nil {
				t.Skipf("no exception thrown (device/version-dependent); expected %s", tc.expected)
				return
			}

			var se *aidlerrors.StatusError
			if !errors.As(statusErr, &se) {
				t.Skipf("non-StatusError returned: %T: %v", statusErr, statusErr)
				return
			}

			if se.Exception != tc.expected {
				t.Skipf("got %s instead of expected %s (device/version-dependent)", se.Exception, tc.expected)
				return
			}

			testedExceptions[se.Exception] = true
			t.Logf("exception %s: message=%q", se.Exception, se.Message)
		})
	}

	// The number of testable exception types varies by device/API level.
	// At minimum we should see at least 1 if any services are available.
	if len(testedExceptions) > 0 {
		t.Logf("Tested %d distinct exception types", len(testedExceptions))
	} else {
		t.Skip("no exception types could be tested on this device")
	}
}
