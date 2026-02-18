package com.syncclipboard.systemhook;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;

import de.robv.android.xposed.XposedBridge;

import org.json.JSONObject;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.ByteArrayOutputStream;
import java.lang.reflect.Method;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.StandardCopyOption;
import java.util.Arrays;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;

final class HookProtocol {
    private static final File HOOK_DIR = new File("/data/system/syncclipboard_hook");
    private static final File EVENT_FILE = new File(HOOK_DIR, "hook_events.log");
    private static final File STATE_FILE = new File(HOOK_DIR, "clipboard_state.json");
    private static final File COMMAND_FILE = new File(HOOK_DIR, "clipboard_command.json");
    private static final File ACK_FILE = new File(HOOK_DIR, "clipboard_ack.json");

    private static final AtomicReference<String> LAST_REQUEST_ID = new AtomicReference<>("");
    private static final AtomicBoolean APPLYING_COMMAND = new AtomicBoolean(false);
    private static final AtomicReference<SetInvocationTemplate> LAST_SET_TEMPLATE = new AtomicReference<>(null);

    private HookProtocol() {
    }

    static void writeState(String content) {
        try {
            ensureDir();
            JSONObject payload = new JSONObject();
            payload.put("content", content == null ? "" : content);
            payload.put("timestamp", System.currentTimeMillis() / 1000);
            writeAtomic(STATE_FILE, payload.toString());
        } catch (Throwable ignored) {
            XposedBridge.log("[SyncClipboardHook] writeState failed: " + ignored);
        }
    }

    static void recordEvent(String event) {
        try {
            ensureDir();

            JSONObject payload = new JSONObject();
            payload.put("event", event);
            payload.put("timestamp", System.currentTimeMillis());

            try (FileOutputStream fos = new FileOutputStream(EVENT_FILE, true)) {
                fos.write(payload.toString().getBytes(StandardCharsets.UTF_8));
                fos.write('\n');
            }
        } catch (Throwable ignored) {
            XposedBridge.log("[SyncClipboardHook] recordEvent failed: " + ignored);
        }
    }

    static void processPendingCommand(Object clipboardServiceInstance) {
        try {
            if (APPLYING_COMMAND.get()) {
                return;
            }

            ensureDir();
            if (!COMMAND_FILE.exists()) {
                return;
            }

            String raw = readAll(COMMAND_FILE);
            if (raw == null || raw.trim().isEmpty()) {
                return;
            }

            JSONObject cmd = new JSONObject(raw);
            String requestId = cmd.optString("request_id", "");
            String action = cmd.optString("action", "");
            String content = cmd.optString("content", "");

            if (requestId.isEmpty() || action.isEmpty()) {
                return;
            }

            if (requestId.equals(LAST_REQUEST_ID.get())) {
                return;
            }

            if ("set".equals(action)) {
                applySetCommand(clipboardServiceInstance, content, requestId);
            }
        } catch (Throwable ignored) {
            XposedBridge.log("[SyncClipboardHook] processPendingCommand failed: " + ignored);
        }
    }

    static boolean isApplyingCommand() {
        return APPLYING_COMMAND.get();
    }

    static void captureSetInvocation(String methodName, Object[] args) {
        if (APPLYING_COMMAND.get()) {
            return;
        }
        if (methodName == null || methodName.isEmpty() || args == null || args.length == 0) {
            return;
        }

        int clipArgIndex = -1;
        for (int i = 0; i < args.length; i++) {
            if (args[i] instanceof ClipData) {
                clipArgIndex = i;
                break;
            }
        }
        if (clipArgIndex < 0) {
            return;
        }

        Object[] copied = Arrays.copyOf(args, args.length);
        LAST_SET_TEMPLATE.set(new SetInvocationTemplate(methodName, copied, clipArgIndex));
    }

    private static void applySetCommand(Object clipboardServiceInstance, String content, String requestId) {
        if (!APPLYING_COMMAND.compareAndSet(false, true)) {
            return;
        }
        try {
            // Mark request as handled first to prevent recursive re-entry
            LAST_REQUEST_ID.set(requestId);

            String normalized = content == null ? "" : content;

            boolean appliedViaTemplate = applyWithCapturedTemplate(clipboardServiceInstance, normalized);
            if (!appliedViaTemplate) {
                applyViaClipboardManager(clipboardServiceInstance, normalized);
            }

            writeAck(requestId, "ok", "");
            //noinspection ResultOfMethodCallIgnored
            COMMAND_FILE.delete();
        } catch (Throwable t) {
            writeAck(requestId, "error", String.valueOf(t));
        } finally {
            APPLYING_COMMAND.set(false);
        }
    }

    private static boolean applyWithCapturedTemplate(Object clipboardServiceInstance, String content) {
        SetInvocationTemplate template = LAST_SET_TEMPLATE.get();
        if (template == null) {
            return false;
        }

        Method candidate = findCompatibleSetMethod(clipboardServiceInstance.getClass(), template);
        if (candidate == null) {
            return false;
        }

        Object[] invokeArgs = Arrays.copyOf(template.args, template.args.length);
        invokeArgs[template.clipArgIndex] = ClipData.newPlainText("text", content);
        try {
            candidate.setAccessible(true);
            candidate.invoke(clipboardServiceInstance, invokeArgs);
            return true;
        } catch (Throwable t) {
            XposedBridge.log("[SyncClipboardHook] template invoke failed: " + t);
            return false;
        }
    }

    private static Method findCompatibleSetMethod(Class<?> serviceClass, SetInvocationTemplate template) {
        Method[] methods = serviceClass.getDeclaredMethods();
        for (Method method : methods) {
            if (!template.methodName.equals(method.getName())) {
                continue;
            }
            Class<?>[] parameterTypes = method.getParameterTypes();
            if (parameterTypes.length != template.args.length) {
                continue;
            }
            if (template.clipArgIndex < 0 || template.clipArgIndex >= parameterTypes.length) {
                continue;
            }
            if (!ClipData.class.isAssignableFrom(parameterTypes[template.clipArgIndex])) {
                continue;
            }
            return method;
        }
        return null;
    }

    private static void applyViaClipboardManager(Object clipboardServiceInstance, String content) throws Exception {
        Object contextObj = de.robv.android.xposed.XposedHelpers.getObjectField(clipboardServiceInstance, "mContext");
        if (!(contextObj instanceof Context)) {
            throw new IllegalStateException("missing service context");
        }

        Context context = (Context) contextObj;
        ClipboardManager manager = (ClipboardManager) context.getSystemService(Context.CLIPBOARD_SERVICE);
        if (manager == null) {
            throw new IllegalStateException("clipboard manager unavailable");
        }

        ClipData data = ClipData.newPlainText("text", content);
        manager.setPrimaryClip(data);
    }

    private static final class SetInvocationTemplate {
        final String methodName;
        final Object[] args;
        final int clipArgIndex;

        SetInvocationTemplate(String methodName, Object[] args, int clipArgIndex) {
            this.methodName = methodName;
            this.args = args;
            this.clipArgIndex = clipArgIndex;
        }
    }

    private static void writeAck(String requestId, String status, String err) {
        try {
            JSONObject ack = new JSONObject();
            ack.put("request_id", requestId);
            ack.put("status", status);
            ack.put("error", err == null ? "" : err);
            ack.put("timestamp", System.currentTimeMillis() / 1000);
            writeAtomic(ACK_FILE, ack.toString());
        } catch (Throwable ignored) {
            XposedBridge.log("[SyncClipboardHook] writeAck failed: " + ignored);
        }
    }

    private static void ensureDir() {
        if (!HOOK_DIR.exists()) {
            //noinspection ResultOfMethodCallIgnored
            HOOK_DIR.mkdirs();
        }
    }

    private static String readAll(File file) {
        try (FileInputStream in = new FileInputStream(file)) {
            ByteArrayOutputStream out = new ByteArrayOutputStream();
            byte[] buffer = new byte[4096];
            int n;
            while ((n = in.read(buffer)) != -1) {
                out.write(buffer, 0, n);
            }
            return new String(out.toByteArray(), StandardCharsets.UTF_8);
        } catch (Throwable e) {
            return null;
        }
    }

    private static void writeAtomic(File target, String content) {
        try {
            File tmp = new File(target.getParentFile(), target.getName() + ".tmp");
            try (FileOutputStream fos = new FileOutputStream(tmp, false)) {
                fos.write(content.getBytes(StandardCharsets.UTF_8));
            }
            Files.move(tmp.toPath(), target.toPath(), StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE);
        } catch (Throwable ignored) {
        }
    }
}
