package com.syncclipboard.systemhook;

import android.content.ClipData;

import de.robv.android.xposed.IXposedHookLoadPackage;
import de.robv.android.xposed.XC_MethodHook;
import de.robv.android.xposed.XposedBridge;
import de.robv.android.xposed.XposedHelpers;
import de.robv.android.xposed.callbacks.XC_LoadPackage;

import java.lang.reflect.Method;
import java.util.HashSet;
import java.util.Set;
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

            dumpClipboardServiceMethods(serviceClass);

            hookSetMethods(serviceClass);
            hookGetMethods(serviceClass);

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

    private static void hookSetMethods(Class<?> serviceClass) {
        Set<String> candidateNames = new HashSet<>();
        candidateNames.add("setPrimaryClip");
        candidateNames.add("setPrimaryClipInternal");
        candidateNames.add("setPrimaryClipAsPackage");

        for (Method m : serviceClass.getDeclaredMethods()) {
            String n = m.getName();
            if (n != null && n.toLowerCase().contains("setprimary") && n.toLowerCase().contains("clip")) {
                candidateNames.add(n);
            }
        }

        for (String name : candidateNames) {
            XposedBridge.hookAllMethods(serviceClass, name, new XC_MethodHook() {
                @Override
                protected void beforeHookedMethod(MethodHookParam param) {
                    SERVICE_REF.set(param.thisObject);
                    HookProtocol.recordEvent("set_hook:" + name);
                    ClipData data = extractClipDataFromArgs(param.args);
                    if (data != null) {
                        HookProtocol.writeState(readClipText(data));
                    }
                    HookProtocol.processPendingCommand(param.thisObject);
                    ensureWorker();
                }
            });
        }
    }

    private static void hookGetMethods(Class<?> serviceClass) {
        Set<String> candidateNames = new HashSet<>();
        candidateNames.add("getPrimaryClip");
        candidateNames.add("getPrimaryClipInternal");
        candidateNames.add("getPrimaryClipAsPackage");

        for (Method m : serviceClass.getDeclaredMethods()) {
            String n = m.getName();
            if (n != null && n.toLowerCase().contains("getprimary") && n.toLowerCase().contains("clip")) {
                candidateNames.add(n);
            }
        }

        for (String name : candidateNames) {
            XposedBridge.hookAllMethods(serviceClass, name, new XC_MethodHook() {
                @Override
                protected void afterHookedMethod(MethodHookParam param) {
                    SERVICE_REF.set(param.thisObject);
                    HookProtocol.recordEvent("get_hook:" + name);
                    Object result = param.getResult();
                    if (result instanceof ClipData) {
                        HookProtocol.writeState(readClipText((ClipData) result));
                    }
                    HookProtocol.processPendingCommand(param.thisObject);
                    ensureWorker();
                }
            });
        }
    }

    private static ClipData extractClipDataFromArgs(Object[] args) {
        if (args == null) {
            return null;
        }
        for (Object arg : args) {
            if (arg instanceof ClipData) {
                return (ClipData) arg;
            }
        }
        return null;
    }

    private static void dumpClipboardServiceMethods(Class<?> serviceClass) {
        try {
            for (Method m : serviceClass.getDeclaredMethods()) {
                String n = m.getName();
                if (n != null && n.toLowerCase().contains("clip")) {
                    XposedBridge.log("[SyncClipboardHook] method: " + m.toString());
                }
            }
        } catch (Throwable ignored) {
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
