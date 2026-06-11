import streamlit as st
import torch
import timm
import numpy as np
from torchvision import transforms
from PIL import Image
from pytorch_grad_cam import GradCAM
from pytorch_grad_cam.utils.image import show_cam_on_image
from pytorch_grad_cam.utils.model_targets import ClassifierOutputTarget

# ── Config ──────────────────────────────────────────────────
CLASS_NAMES = [
    'AnnualCrop', 'Forest', 'HerbaceousVegetation', 'Highway',
    'Industrial', 'Pasture', 'PermanentCrop', 'Residential',
    'River', 'SeaLake'
]
CLASS_EMOJI = {
    'AnnualCrop': '🌾', 'Forest': '🌲', 'HerbaceousVegetation': '🌿',
    'Highway': '🛣️', 'Industrial': '🏭', 'Pasture': '🐄',
    'PermanentCrop': '🍇', 'Residential': '🏘️', 'River': '🌊',
    'SeaLake': '🏖️'
}
DEVICE = torch.device("cpu")

# ── Load Model (cached) ──────────────────────────────────────
@st.cache_resource
def load_model():
    model = timm.create_model('efficientnet_b0', pretrained=False, num_classes=10)
    model.load_state_dict(torch.load('eurosat_efficientnet.pth', map_location=DEVICE))
    model.eval()
    return model

# ── Transforms ───────────────────────────────────────────────
transform = transforms.Compose([
    transforms.Resize((64, 64)),
    transforms.ToTensor(),
    transforms.Normalize(mean=[0.3444, 0.3803, 0.4078],
                         std=[0.2034, 0.1366, 0.1153])
])
inv_norm = transforms.Normalize(
    mean=[-0.3444/0.2034, -0.3803/0.1366, -0.4078/0.1153],
    std=[1/0.2034, 1/0.1366, 1/0.1153]
)

# ── Predict + GradCAM ────────────────────────────────────────
def predict(model, img: Image.Image):
    tensor = transform(img.convert("RGB")).unsqueeze(0)

    with torch.no_grad():
        logits = model(tensor)
        probs  = torch.softmax(logits, dim=1)[0]

    pred_idx  = probs.argmax().item()
    pred_class = CLASS_NAMES[pred_idx]
    confidence = probs[pred_idx].item()

    # GradCAM
    cam        = GradCAM(model=model, target_layers=[model.conv_head])
    gcam       = cam(input_tensor=tensor,
                     targets=[ClassifierOutputTarget(pred_idx)])[0]
    rgb        = inv_norm(tensor[0]).permute(1, 2, 0).numpy().clip(0, 1)
    overlay    = show_cam_on_image(rgb.astype(np.float32), gcam, use_rgb=True)

    return pred_class, confidence, probs.numpy(), overlay

# ── UI ───────────────────────────────────────────────────────
st.set_page_config(
    page_title="Satellite Land Use Classifier",
    page_icon="🛰️",
    layout="wide"
)

st.title("🛰️ Satellite Land Use Classifier")
st.caption("EfficientNet-B0 trained on EuroSAT · 95.7% accuracy · 10 land-use classes")
st.divider()

model = load_model()

col_upload, col_result = st.columns([1, 1], gap="large")

with col_upload:
    st.subheader("Upload a satellite image")
    uploaded = st.file_uploader(
        "JPG or PNG, ideally 64×64 satellite imagery",
        type=["jpg", "jpeg", "png"]
    )

    st.markdown("**Or try a sample class:**")
    sample_cols = st.columns(5)
    samples = list(CLASS_EMOJI.items())
    sample_choice = None
    for i, (cls, emoji) in enumerate(samples):
        with sample_cols[i % 5]:
            if st.button(f"{emoji}", help=cls, use_container_width=True):
                sample_choice = cls

    if sample_choice:
        st.info(f"Selected: **{sample_choice}** — upload an actual image of this class to test.")

with col_result:
    if uploaded:
        img = Image.open(uploaded).convert("RGB")

        with st.spinner("Classifying..."):
            pred_class, confidence, all_probs, gradcam_img = predict(model, img)

        st.subheader("Results")

        r1, r2 = st.columns(2)
        with r1:
            st.image(img, caption="Input image", use_container_width=True)
        with r2:
            st.image(gradcam_img, caption="Grad-CAM heatmap", use_container_width=True)

        emoji = CLASS_EMOJI[pred_class]
        st.markdown(f"### {emoji} {pred_class}")

        conf_color = "green" if confidence > 0.85 else "orange" if confidence > 0.6 else "red"
        st.markdown(f"**Confidence:** :{conf_color}[{confidence:.1%}]")
        st.progress(float(confidence))

        st.markdown("**All class probabilities:**")
        for cls, prob in sorted(zip(CLASS_NAMES, all_probs),
                                key=lambda x: x[1], reverse=True):
            bar_width = int(prob * 100)
            highlight = "**" if cls == pred_class else ""
            st.markdown(f"{CLASS_EMOJI[cls]} {highlight}{cls}{highlight}")
            st.progress(float(prob), text=f"{prob:.1%}")

    else:
        st.info("👈 Upload a satellite image to get started")
        st.markdown("**Model details:**")
        st.markdown("""
        - Architecture: EfficientNet-B0
        - Dataset: EuroSAT (27,000 images)
        - Test accuracy: **95.7%**
        - Training: Transfer learning from ImageNet
        - Interpretability: Grad-CAM heatmaps
        """)
