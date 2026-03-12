package com.syncclipboard.helper;

import android.app.Service;
import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.content.Intent;
import android.os.IBinder;
import android.util.Log;

import java.io.File;
import java.io.FileWriter;
import java.util.Base64;

/**
 * 剪贴板辅助服务
 * 通过 Intent + 文件结果协议与 Go 服务交互
 * 
 * 通信协议：
 * - 输入参数：Intent extra `op` / `text_b64`
 * - 数据文件：/data/local/tmp/clipboard_data.txt
 * - 状态文件：/data/local/tmp/clipboard_status.txt
 * 
 * 命令格式：
 * - op=get
 * - op=set + text_b64
 */
public class ClipboardService extends Service {
    private static final String TAG = "SyncClipboard";
    private static final String DATA_FILE = "/data/user/0/com.syncclipboard.helper/files/clipboard_data.txt";
    private static final String STATUS_FILE = "/data/user/0/com.syncclipboard.helper/files/clipboard_status.txt";
    
    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        final Intent finalIntent = intent;
        new Thread(() -> {
            try {
                processCommand(finalIntent);
            } catch (Exception e) {
                Log.e(TAG, "Error processing clipboard command", e);
                writeStatus("error: " + e.getMessage());
            } finally {
                stopSelf();
            }
        }).start();
        
        return START_NOT_STICKY;
    }
    
    private void processCommand(Intent intent) throws Exception {
        if (intent == null) {
            writeStatus("error: empty intent");
            return;
        }

        String cmd = intent.getStringExtra("op");
        if (cmd == null || cmd.trim().isEmpty()) {
            writeStatus("error: empty command");
            return;
        }

        cmd = cmd.trim();
        Log.d(TAG, "Processing command: " + cmd);
        
        ClipboardManager clipboard = (ClipboardManager) getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard == null) {
            writeStatus("error: clipboard service not available");
            return;
        }
        
        if ("get".equals(cmd)) {
            // 读取剪贴板
            if (!clipboard.hasPrimaryClip()) {
                writeStatus("ok");
                writeData("");
                return;
            }
            
            ClipData clip = clipboard.getPrimaryClip();
            if (clip == null || clip.getItemCount() == 0) {
                writeStatus("ok");
                writeData("");
                return;
            }
            
            CharSequence text = clip.getItemAt(0).getText();
            String content = (text != null) ? text.toString() : "";
            
            writeData(content);
            writeStatus("ok");
            Log.d(TAG, "Read clipboard: " + content.length() + " bytes");
            
        } else if ("set".equals(cmd)) {
            String contentB64 = intent.getStringExtra("text_b64");
            if (contentB64 == null) {
                writeStatus("error: missing text_b64");
                return;
            }

            String content;
            try {
                content = new String(Base64.getDecoder().decode(contentB64));
            } catch (IllegalArgumentException e) {
                writeStatus("error: invalid base64 content");
                return;
            }

            ClipData clipData = ClipData.newPlainText("text", content);
            clipboard.setPrimaryClip(clipData);
            
            writeStatus("ok");
            Log.d(TAG, "Wrote clipboard: " + content.length() + " bytes");
            
        } else {
            writeStatus("error: unknown command: " + cmd);
        }
    }
    
    private void writeData(String content) throws Exception {
        FileWriter writer = new FileWriter(DATA_FILE);
        writer.write(content);
        writer.close();
        
        // 确保文件权限
        new File(DATA_FILE).setReadable(true, false);
        new File(DATA_FILE).setWritable(true, false);
    }
    
    private void writeStatus(String status) {
        try {
            FileWriter writer = new FileWriter(STATUS_FILE);
            writer.write(status);
            writer.close();
            
            // 确保文件权限
            new File(STATUS_FILE).setReadable(true, false);
            new File(STATUS_FILE).setWritable(true, false);
        } catch (Exception e) {
            Log.e(TAG, "Failed to write status", e);
        }
    }
    
    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
