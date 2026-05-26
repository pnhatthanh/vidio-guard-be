"""
CustomPhoBERT — must match training architecture (Kaggle/Colab).
CLS + mean pooling → 2-layer GELU classifier.
"""
import os

import torch
import torch.nn as nn
from transformers import AutoModel


class CustomPhoBERT(nn.Module):
    def __init__(self, model_name: str, num_labels: int, dropout_prob: float, unfreeze_last_n: int = 4):
        super().__init__()
        self.phobert = AutoModel.from_pretrained(model_name)
        self.config = self.phobert.config
        hidden_size = self.config.hidden_size

        for param in self.phobert.parameters():
            param.requires_grad = False

        encoder_layers = self.phobert.encoder.layer
        for layer in encoder_layers[-unfreeze_last_n:]:
            for param in layer.parameters():
                param.requires_grad = True

        if hasattr(self.phobert, "pooler") and self.phobert.pooler is not None:
            for param in self.phobert.pooler.parameters():
                param.requires_grad = True

        self.classifier = nn.Sequential(
            nn.Linear(hidden_size * 2, hidden_size // 2),
            nn.GELU(),
            nn.Dropout(dropout_prob),
            nn.Linear(hidden_size // 2, num_labels),
        )

    def forward(self, input_ids, attention_mask, **kwargs):
        outputs = self.phobert(
            input_ids=input_ids,
            attention_mask=attention_mask,
            **kwargs,
        )
        last_hidden_state = outputs.last_hidden_state

        cls_output = last_hidden_state[:, 0, :]

        mask_expanded = attention_mask.unsqueeze(-1).expand(last_hidden_state.size()).float()
        sum_embeddings = torch.sum(last_hidden_state * mask_expanded, dim=1)
        sum_mask = torch.clamp(mask_expanded.sum(dim=1), min=1e-9)
        mean_pooled = sum_embeddings / sum_mask

        pooled = torch.cat([cls_output, mean_pooled], dim=-1)
        logits = self.classifier(pooled)

        class Output:
            def __init__(self, logits):
                self.logits = logits

        return Output(logits)

    @classmethod
    def from_pretrained(
        cls,
        save_directory: str,
        base_model_name: str,
        num_labels: int,
        dropout_prob: float,
        unfreeze_last_n: int = 4,
    ) -> "CustomPhoBERT":
        model = cls(base_model_name, num_labels, dropout_prob, unfreeze_last_n)
        state_dict_path = os.path.join(save_directory, "pytorch_model.bin")
        if not os.path.exists(state_dict_path):
            raise FileNotFoundError(f"weights not found: {state_dict_path}")

        state_dict = torch.load(state_dict_path, map_location="cpu", weights_only=False)
        model.load_state_dict(state_dict)
        return model
