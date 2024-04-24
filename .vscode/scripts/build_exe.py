import json
import os
import winsound

# 環境変数 WORKSPACE_FOLDER の値を取得
workspace_folder = os.environ.get('WORKSPACE_FOLDER')

# 値を表示
print(f"workspace_folder: {workspace_folder}")

# Build command
# -o 出力フォルダ
# -trimpath ビルドパスを削除
# -v ビルドログを出力
# -a 全ての依存関係を再ビルド
# -buildmode=exe 実行可能ファイルを生成
# -ldflags "-s -w" バイナリサイズを小さくする
# -gcflags "all=-N -l" デバッグ情報を削除
build_command = f"cd go && go build -o {workspace_folder}/build/mat4.exe -trimpath " \
                f"-v -a -buildmode=exe -ldflags \"-s -w\" " \
                f"{workspace_folder}/go/cmd/main.go"

print(f"build_command: {build_command}")

os.system(build_command)

# Play beep sound
winsound.PlaySound("SystemAsterisk", winsound.SND_ALIAS)
