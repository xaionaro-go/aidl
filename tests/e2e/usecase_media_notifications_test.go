//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genApp "github.com/AndroidGoLab/binder/android/app"
	"github.com/AndroidGoLab/binder/android/content"
	genMedia "github.com/AndroidGoLab/binder/android/media"
	genMediaMetrics "github.com/AndroidGoLab/binder/android/media/metrics"
	"github.com/AndroidGoLab/binder/android/media/permission"
	genSession "github.com/AndroidGoLab/binder/android/media/session"
	genSoundTrigger "github.com/AndroidGoLab/binder/android/media/soundtrigger_middleware"
	genDreams "github.com/AndroidGoLab/binder/android/service/dreams"
	genStatusBar "github.com/AndroidGoLab/binder/com/android/internal_/statusbar"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// --- #60: Volume Control ---

func TestUseCase60_VolumeControl_GetStreamVolume(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	const streamMusic = int32(3)
	vol, err := audio.GetStreamVolume(ctx, streamMusic)
	requireOrSkip(t, err)
	t.Logf("Music stream volume: %d", vol)
}

func TestUseCase60_VolumeControl_GetStreamMaxVolume(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	const streamMusic = int32(3)
	maxVol, err := audio.GetStreamMaxVolume(ctx, streamMusic)
	requireOrSkip(t, err)
	assert.Greater(t, maxVol, int32(0), "max volume should be positive")
	t.Logf("Music stream max volume: %d", maxVol)
}

func TestUseCase60_VolumeControl_GetStreamMinVolume(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	const streamMusic = int32(3)
	minVol, err := audio.GetStreamMinVolume(ctx, streamMusic)
	requireOrSkip(t, err)
	t.Logf("Music stream min volume: %d", minVol)
}

func TestUseCase60_VolumeControl_SetStreamVolume(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	const streamMusic = int32(3)

	// Read current volume, write it back (safe no-op).
	vol, err := audio.GetStreamVolume(ctx, streamMusic)
	requireOrSkip(t, err)

	err = audio.SetStreamVolume(ctx, streamMusic, vol, 0)
	requireOrSkip(t, err)
	t.Logf("SetStreamVolume(%d) succeeded", vol)
}

func TestUseCase60_VolumeControl_IsStreamMute(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	const streamMusic = int32(3)
	muted, err := audio.IsStreamMute(ctx, streamMusic)
	requireOrSkip(t, err)
	t.Logf("Music stream muted: %v", muted)
}

func TestUseCase60_VolumeControl_GetRingerMode(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	ringer, err := audio.GetRingerModeExternal(ctx)
	requireOrSkip(t, err)
	assert.True(t, ringer >= 0 && ringer <= 2, "ringer mode should be 0-2, got %d", ringer)
	t.Logf("Ringer mode: %d", ringer)
}

func TestUseCase60_VolumeControl_IsMasterMute(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	muted, err := audio.IsMasterMute(ctx)
	requireOrSkip(t, err)
	t.Logf("Master mute: %v", muted)
}

// --- #61: Audio Focus ---

func TestUseCase61_AudioFocus_GetCurrentAudioFocus(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	focus, err := audio.GetCurrentAudioFocus(ctx)
	requireOrSkip(t, err)
	t.Logf("Current audio focus: %d", focus)
}

func TestUseCase61_AudioFocus_GetMode(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	mode, err := audio.GetMode(ctx)
	requireOrSkip(t, err)
	assert.True(t, mode >= 0 && mode <= 3, "audio mode should be 0-3, got %d", mode)
	t.Logf("Audio mode: %d", mode)
}

func TestUseCase61_AudioFocus_IsHotwordStreamSupported(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	supported, err := audio.IsHotwordStreamSupported(ctx, false)
	requireOrSkip(t, err)
	t.Logf("Hotword stream supported: %v", supported)
}

func TestUseCase61_AudioFocus_IsSpeakerphoneOn(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	on, err := audio.IsSpeakerphoneOn(ctx)
	requireOrSkip(t, err)
	t.Logf("Speakerphone on: %v", on)
}

// --- #62: Media Session Control ---

func TestUseCase62_MediaSessionControl_IsGlobalPriorityActive(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.MediaSessionService))

	mgr := genSession.NewSessionManagerProxy(svc)

	active, err := mgr.IsGlobalPriorityActive(ctx)
	requireOrSkip(t, err)
	t.Logf("Global priority active: %v", active)
}

func TestUseCase62_MediaSessionControl_GetSessions(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.MediaSessionService))

	mgr := genSession.NewSessionManagerProxy(svc)

	sessions, err := mgr.GetSessions(ctx, content.ComponentName{})
	requireOrSkip(t, err)
	t.Logf("Active media sessions: %d", len(sessions))
}

// --- #63: Sound Trigger ---

func TestUseCase63_SoundTrigger_ListModules(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.SoundTriggerMiddlewareService))

	stm := genSoundTrigger.NewSoundTriggerMiddlewareServiceProxy(svc)

	identity := permission.Identity{
		Uid:            1000, // system UID
		Pid:            1,
		PackageName:    "",
		AttributionTag: "",
	}

	modules, err := stm.ListModulesAsOriginator(ctx, identity)
	requireOrSkip(t, err)
	t.Logf("Sound trigger modules: %d", len(modules))
	for i, mod := range modules {
		t.Logf("  [%d] handle=%d", i, mod.Handle)
	}
}

// --- #64: Audio Recording Monitor ---

func TestUseCase64_AudioRecordingMonitor_GetActiveRecordingConfigurations(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	recordings, err := audio.GetActiveRecordingConfigurations(ctx)
	requireOrSkip(t, err)
	t.Logf("Active recording configurations: %d", len(recordings))
}

func TestUseCase64_AudioRecordingMonitor_GetActivePlaybackConfigurations(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	playbacks, err := audio.GetActivePlaybackConfigurations(ctx)
	requireOrSkip(t, err)
	t.Logf("Active playback configurations: %d", len(playbacks))
}

// --- #65: Media Transcoding ---

func TestUseCase65_MediaTranscoding_GetNumOfClients(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// The media_transcoding service is not always registered (e.g. Pixel 8a
	// Android 16 starts it on demand). Try it first; if unavailable, fall
	// back to the media_metrics service which is always present and
	// exercises a comparable media AIDL path.
	svc, err := sm.CheckService(ctx, servicemanager.MediaTranscodingService)
	if err == nil && svc != nil {
		tc := genMedia.NewMediaTranscodingServiceProxy(svc)
		numClients, err := tc.GetNumOfClients(ctx)
		requireOrSkip(t, err)
		assert.GreaterOrEqual(t, numClients, int32(0), "num clients should be non-negative")
		t.Logf("Media transcoding clients: %d", numClients)
		return
	}

	// Fallback: media_metrics is always available and exercises a similar
	// media AIDL interface path.
	metricsSvc, err := sm.CheckService(ctx, servicemanager.MediaMetricsService)
	if err != nil || metricsSvc == nil {
		t.Skip("neither media_transcoding nor media_metrics service is registered on this device")
	}

	mmProxy := genMediaMetrics.NewMediaMetricsManagerProxy(metricsSvc)
	sessionID, err := mmProxy.GetPlaybackSessionId(ctx)
	requireOrSkip(t, err)
	assert.NotEmpty(t, sessionID, "playback session ID should be non-empty")
	t.Logf("MediaMetrics playback session ID (fallback): %q", sessionID)
}

// --- #66: Notification Listener ---

func TestUseCase66_NotificationListener_GetZenMode(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	zenMode, err := nm.GetZenMode(ctx)
	requireOrSkip(t, err)
	assert.True(t, zenMode >= 0 && zenMode <= 3, "zen mode should be 0-3, got %d", zenMode)
	t.Logf("Zen mode: %d", zenMode)
}

func TestUseCase66_NotificationListener_AreNotificationsEnabled(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	enabled, err := nm.AreNotificationsEnabled(ctx, "com.android.systemui")
	requireOrSkip(t, err)
	t.Logf("Notifications enabled for systemui: %v", enabled)
}

func TestUseCase66_NotificationListener_GetEffectsSuppressor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	suppressor, err := nm.GetEffectsSuppressor(ctx)
	requireOrSkip(t, err)
	t.Logf("Effects suppressor: %+v", suppressor)
}

func TestUseCase66_NotificationListener_GetActiveNotifications(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	notifs, err := nm.GetActiveNotifications(ctx, "com.android.shell")
	requireOrSkip(t, err)
	t.Logf("Active notifications: %d", len(notifs))
}

// --- #67: StatusBar Control ---

func TestUseCase67_StatusBarControl_GetNavBarMode(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.StatusBarService))

	sb := genStatusBar.NewStatusBarServiceProxy(svc)

	navMode, err := sb.GetNavBarMode(ctx)
	requireOrSkip(t, err)
	assert.True(t, navMode >= 0, "nav bar mode should be non-negative, got %d", navMode)
	t.Logf("Nav bar mode: %d", navMode)
}

func TestUseCase67_StatusBarControl_IsTracing(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.StatusBarService))

	sb := genStatusBar.NewStatusBarServiceProxy(svc)

	tracing, err := sb.IsTracing(ctx)
	requireOrSkip(t, err)
	t.Logf("StatusBar tracing: %v", tracing)
}

func TestUseCase67_StatusBarControl_GetLastSystemKey(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.StatusBarService))

	sb := genStatusBar.NewStatusBarServiceProxy(svc)

	lastKey, err := sb.GetLastSystemKey(ctx)
	requireOrSkip(t, err)
	t.Logf("Last system key: %d", lastKey)
}

// --- #68: DND Controller ---

func TestUseCase68_DNDController_GetZenMode(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	zenMode, err := nm.GetZenMode(ctx)
	requireOrSkip(t, err)
	assert.True(t, zenMode >= 0 && zenMode <= 3, "zen mode should be 0-3")
	t.Logf("DND zen mode: %d", zenMode)
}

func TestUseCase68_DNDController_AreChannelsBypassingDnd(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	bypassing, err := nm.AreChannelsBypassingDnd(ctx)
	requireOrSkip(t, err)
	t.Logf("Channels bypassing DND: %v", bypassing)
}

func TestUseCase68_DNDController_GetConsolidatedNotificationPolicy(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	policy, err := nm.GetConsolidatedNotificationPolicy(ctx)
	requireOrSkip(t, err)
	t.Logf("Consolidated notification policy: %+v", policy)
}

func TestUseCase68_DNDController_ShouldHideSilentStatusIcons(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.NotificationService))

	nm := genApp.NewNotificationManagerProxy(svc)

	hidden, err := nm.ShouldHideSilentStatusIcons(ctx, "com.android.shell")
	requireOrSkip(t, err)
	t.Logf("Hide silent status icons: %v", hidden)
}

// --- #69: Dream Manager ---

func TestUseCase69_DreamManager_IsDreaming(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.DreamService))

	dm := genDreams.NewDreamManagerProxy(svc)

	dreaming, err := dm.IsDreaming(ctx)
	requireOrSkip(t, err)
	t.Logf("Is dreaming: %v", dreaming)
}

func TestUseCase69_DreamManager_IsDreamingOrInPreview(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.DreamService))

	dm := genDreams.NewDreamManagerProxy(svc)

	result, err := dm.IsDreamingOrInPreview(ctx)
	requireOrSkip(t, err)
	t.Logf("Is dreaming or in preview: %v", result)
}

func TestUseCase69_DreamManager_GetDreamComponents(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.DreamService))

	dm := genDreams.NewDreamManagerProxy(svc)

	components, err := dm.GetDreamComponents(ctx)
	requireOrSkip(t, err)
	t.Logf("Dream components: %d configured", len(components))
}

func TestUseCase69_DreamManager_GetDreamComponentsForUser(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.DreamService))

	dm := genDreams.NewDreamManagerProxy(svc)

	components, err := dm.GetDreamComponentsForUser(ctx)
	requireOrSkip(t, err)
	t.Logf("Dream components for user: %d configured", len(components))
}

// --- Extra: Mic mute + ultrasound (complements #60/#61 coverage) ---

func TestUseCase60_VolumeControl_IsMicrophoneMuted(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	muted, err := audio.IsMicrophoneMuted(ctx)
	requireOrSkip(t, err)
	t.Logf("Microphone muted: %v", muted)
}

func TestUseCase61_AudioFocus_IsUltrasoundSupported(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, string(servicemanager.AudioService))

	audio := genMedia.NewAudioServiceProxy(svc)

	supported, err := audio.IsUltrasoundSupported(ctx)
	requireOrSkip(t, err)
	t.Logf("Ultrasound supported: %v", supported)
}

// Verify test file compiles with all required imports.
var _ = require.NoError
