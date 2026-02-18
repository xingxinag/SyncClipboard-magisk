package com.syncclipboard.systemhook;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;

import org.json.JSONObject;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.ByteArrayOutputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.StandardCopyOption;
import java.util.concurrent.atomic.AtomicReference;

final class HookProtocol {
    private static final File HOOK_DIR = new File("/data/adb/syncclipboard/hook");
    private static final File EVENT_FILE = new File(HOOK_DIR, "hook_events.log");
    private static final File STATE_FILE = new File(HOOK_DIR, "clipboard_state.json");
    private static final File COMMAND_FILE = new File(HOOK_DIR, "clipboard_command.json");
    private static final File ACK_FILE = new File(HOOK_DIR, "clipboard_ack.json");

    private static final AtomicReference<String> LAST_REQUEST_ID = new AtomicReference<>("");

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
        }
    }

    static void processPendingCommand(Object clipboardServiceInstance) {
        try {
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
        }
    }

    private static void applySetCommand(Object clipboardServiceInstance, String content, String requestId) {
        try {
            Object contextObj = de.robv.android.xposed.XposedHelpers.getObjectField(clipboardServiceInstance, "mContext");
            if (!(contextObj instanceof Context)) {
                writeAck(requestId, "error", "missing service context");
                return;
            }

            Context context = (Context) contextObj;
            ClipboardManager manager = (ClipboardManager) context.getSystemService(Context.CLIPBOARD_SERVICE);
            if (manager == null) {
                writeAck(requestId, "error", "clipboard manager unavailable");
                return;
            }

            ClipData data = ClipData.newPlainText("text", content == null ? "" : content);
            manager.setPrimaryClip(data);

            writeState(content == null ? "" : content);
            writeAck(requestId, "ok", "");
            LAST_REQUEST_ID.set(requestId);
        } catch (Throwable t) {
            writeAck(requestId, "error", String.valueOf(t));
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
