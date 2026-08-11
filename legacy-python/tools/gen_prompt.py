import re, pathlib
src = pathlib.Path('chatbot/prompts/system_prompt.py').read_text(encoding='utf-8')
m = re.search(r'return f"""(.*?)"""\.strip\(\)', src, re.S)
body = m.group(1)

subs = [
    ('{fd["fieldName"]}',       '{{py .fd.FieldName}}'),
    ('{fd["area"]}',            '{{py .fd.Area}}'),
    ('{fd["date"]}',            '{{py .fd.Date}}'),
    ('{fd["confidence"]}',      '{{py .fd.Confidence}}'),
    ('{fd["cleanScenes"]}',     '{{py .fd.CleanScenes}}'),
    ('{cvi:.4f}',               '{{f4 .fd.CVI}}'),
    ('{ndvi:.4f}',              '{{f4 .fd.NDVI}}'),
    ('{evi:.4f}',               '{{f4 .fd.EVI}}'),
    ('{savi:.4f}',              '{{f4 .fd.SAVI}}'),
    ('{ndmi:.4f}',              '{{f4 .fd.NDMI}}'),
    ('{gndvi:.4f}',             '{{f4 .fd.GNDVI}}'),
    ('{hd["stressedPct"]}',     '{{py .hd.StressedPct}}'),
    ('{hd["stressedLocation"]}','{{py .hd.StressedLocation}}'),
    ('{hd["moderatePct"]}',     '{{py .hd.ModeratePct}}'),
    ('{hd["moderateLocation"]}','{{py .hd.ModerateLocation}}'),
    ('{hd["healthyPct"]}',      '{{py .hd.HealthyPct}}'),
    ('{hd["healthyLocation"]}', '{{py .hd.HealthyLocation}}'),
]
out = body
for a, b in subs:
    out = out.replace(a, b)

probe = out.replace('{{', '\x00').replace('}}', '\x01')
leftover = re.findall(r'\{[^{}]*\}', probe)
assert not leftover, "unmapped placeholders: %r" % leftover

BT = chr(96)
escaped = out.replace(BT, BT + ' + "' + BT + '" + ' + BT)

header = [
 'package chatbot',
 '',
 '// rawSystemPrompt is a mechanical transcription of the f-string in',
 '// legacy-python/chatbot/prompts/system_prompt.py, GENERATED rather than',
 '// retyped so that no character can drift. The non-ASCII typography is',
 '// preserved verbatim: U+2019 in "farmer’s", the U+2192 arrows, and the',
 '// U+2013 en dash in "2–3 days".',
 '//',
 '// Placeholders map onto the template funcs `py` (Python f-string rendering)',
 '// and `f4` (Python’s {:.4f}).',
 '//',
 '// Regenerate with: legacy-python/tools/gen_prompt.py',
 'const rawSystemPrompt = ' + BT + escaped + BT,
 '',
]
pathlib.Path('../internal/chatbot/prompt_body.go').write_text('\n'.join(header), encoding='utf-8')
print("generated prompt_body.go,", len(out), "chars")
