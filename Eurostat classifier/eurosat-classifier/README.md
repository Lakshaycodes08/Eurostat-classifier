# 🛰️ Satellite Land Use Classifier

End-to-end computer vision pipeline for classifying satellite imagery 
into 10 land-use categories using transfer learning.

**[Live Demo →](your-streamlit-link-here)**

## Results
| Metric | Score |
|--------|-------|
| Test Accuracy | 95.7% |
| Macro F1 | 0.95 |
| Best class | Forest / SeaLake (F1: 0.98) |
| Hardest class | PermanentCrop (F1: 0.91) |

## Model
- Architecture: EfficientNet-B0 (pretrained on ImageNet)
- Dataset: EuroSAT RGB (27,000 satellite images, 10 classes)
- Training: Two-phase — frozen base (5 epochs) → full fine-tune (10 epochs)
- Interpretability: Grad-CAM heatmaps

## Screenshots
![Training Curves](assets/training_curves.png)
![Confusion Matrix](assets/confusion_matrix.png)
![Grad-CAM](assets/gradcam.png)

## Run Locally
```bash
pip install -r requirements.txt
streamlit run app.py
```
