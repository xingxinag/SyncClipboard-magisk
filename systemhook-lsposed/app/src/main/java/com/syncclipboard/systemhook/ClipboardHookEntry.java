package com.syncclipboard.systemhook;

import android.content.ClipData;

import de.robv.android.xposed.IXposedHookLoadPackage;
import de.robv.android.xposed.XC_MethodHook;
import de.robv.android.xposed.XposedBridge;
import de.robv.android.xposed.XposedHelpers;
import de.robv.android.xposed.callbacks.XC_LoadPackage;

import java.util.concurrent.atomic.AtomicReference;
import java.util.concurrent.atomic.AtomicBoolean;

public class ClipboardHookEntry implements IXposedHookLoadPackage {
    private static final AtomicBoolean WORKER_STARTED = new AtomicBoolean(false);
    private static final AtomicReference<Object> SERVICE_REF = new AtomicReference<>(null);

    @Override
    public void handleLoadPackage(XC_LoadPackage.LoadPackageParam lpparam) {
        if (!"android".equals(lpparam.packageName) || !"android".equals(lpparam.processName)) {
            return;
        }

        try {
            Class<?> serviceClass = XposedHelpers.findClass(
                    "com.android.server.clipboard.ClipboardService",
                    lpparam.classLoader
            );

            XposedBridge.hookAllMethods(serviceClass, "setPrimaryClip", new XC_MethodHook() {
                @Override
                protected void beforeHookedMethod(MethodHookParam param) {
                    SERVICE_REF.set(param.thisObject);
                    HookProtocol.recordEvent("setPrimaryClip");
                    if (param.args != null && param.args.length > 0 && param.args[0] instanceof ClipData) {
                        String content = readClipText((ClipData) param.args[0]);
                        HookProtocol.writeState(content);
                    }
                    HookProtocol.processPendingCommand(param.thisObject);
                    ensureWorker();
                }
            });

            XposedBridge.hookAllMethods(serviceClass, "getPrimaryClip", new XC_MethodHook() {
                @Override
                protected void afterHookedMethod(MethodHookParam param) {
                    SERVICE_REF.set(param.thisObject);
                    HookProtocol.recordEvent("getPrimaryClip");
                    Object result = param.getResult();
                    if (result instanceof ClipData) {
                        HookProtocol.writeState(readClipText((ClipData) result));
                    }
                    HookProtocol.processPendingCommand(param.thisObject);
                    ensureWorker();
                }
            });

            XposedBridge.hookAllConstructors(serviceClass, new XC_MethodHook() {
                @Override
                protected void afterHookedMethod(MethodHookParam param) {
                    SERVICE_REF.set(param.thisObject);
                    HookProtocol.recordEvent("ClipboardService_ctor");
                    ensureWorker();
                }
            });

            XposedBridge.log("[SyncClipboardHook] ClipboardService hooks installed");
        } catch (Throwable t) {
            XposedBridge.log("[SyncClipboardHook] failed to install hooks: " + t);
        }
    }

    private static String readClipText(ClipData data) {
        if (data == null || data.getItemCount() == 0) {
            return "";
        }
        ClipData.Item item = data.getItemAt(0);
        if (item == null) {
            return "";
        }
        CharSequence txt = item.getText();
        return txt == null ? "" : txt.toString();
    }

    private static void ensureWorker() {
        if (!WORKER_STARTED.compareAndSet(false, true)) {
            return;
        }

        Thread worker = new Thread(() -> {
            while (true) {
                try {
                    Object service = SERVICE_REF.get();
                    if (service != null) {
                        HookProtocol.processPendingCommand(service);
                    }
                    Thread.sleep(120);
                } catch (Throwable ignored) {
                }
            }
        }, "SyncClipboardHookWorker");
        worker.setDaemon(true);
        worker.start();
    }
}
