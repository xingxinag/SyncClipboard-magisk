package com.syncclipboard.helper;

import android.app.Activity;
import android.content.ClipboardManager;
import android.content.ClipData;
import android.os.Bundle;
import java.io.FileWriter;
import java.io.BufferedReader;
import java.io.FileReader;

public class MainActivity extends Activity {
    private static final String CLIP_FILE = "/data/local/tmp/clipboard.txt";
    private static final String CMD_FILE = "/data/local/tmp/clipboard_cmd.txt";
    
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        
        try {
            // 读取命令
            BufferedReader reader = new BufferedReader(new FileReader(CMD_FILE));
            String cmd = reader.readLine();
            reader.close();
            
            ClipboardManager clipboard = (ClipboardManager) getSystemService(CLIPBOARD_SERVICE);
            
            if ("get".equals(cmd)) {
                // 读取剪贴板
                if (clipboard.hasPrimaryClip()) {
                    ClipData clip = clipboard.getPrimaryClip();
                    if (clip != null && clip.getItemCount() > 0) {
                        CharSequence text = clip.getItemAt(0).getText();
                        if (text != null) {
                            FileWriter writer = new FileWriter(CLIP_FILE);
                            writer.write(text.toString());
                            writer.close();
                        }
                    }
                }
            } else if (cmd.startsWith("set:")) {
                // 写入剪贴板
                String content = cmd.substring(4);
                ClipData clip = ClipData.newPlainText("text", content);
                clipboard.setPrimaryClip(clip);
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
        
        finish();
    }
}
