# PhoBERT weights (CustomPhoBERT — Clean / Toxic)

Copy your trained model folder here (same layout as Google Drive / Kaggle export):

```
models/
  pytorch_model.bin      # required
  tokenizer_config.json
  vocab.txt
  bpe.codes
  ...
```

Training uses **CustomPhoBERT** (see `app/phobert_arch.py`) with labels `0=Clean`, `1=Toxic`.

Set `AUDIO_PHOBERT_MODEL_PATH=models` (default).
