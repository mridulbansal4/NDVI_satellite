import subprocess

try:
    out = subprocess.check_output(['git', 'show', 'HEAD:frontend/src/index.css'])
    with open('frontend/src/analysis.css', 'wb') as f:
        f.write(out)
    print("Success")
except Exception as e:
    print("Error:", e)
