import gradio as gr
import torch
import timm
import numpy as np
from torchvision import transforms
from PIL import Image
from pytorch_grad_cam import GradCAM
from pytorch_grad_cam.utils.image import show_cam_on_image
from pytorch_grad_cam.utils.model_targets import ClassifierOutputTarget

CLASS_NAMES = ['AnnualCrop','Forest','HerbaceousVegetation','Highway',
               'Industrial','Pasture','PermanentCrop','Residential','River','SeaLake']

DEVICE = torch.device("cpu")

model = timm.create_model('efficientnet_b0', pretrained=False, num_classes=10)
model.load_state_dict(torch.load('eurosat_efficientnet.pth', map_location=DEVICE))
model.eval()

transform = transforms.Compose([
    transforms.Resize((64, 64)),
    transforms.ToTensor(),
    transforms.Normalize(mean=[0.3444, 0.3803, 0.4078], std=[0.2034, 0.1366, 0.1153])
])
inv_norm = transforms.Normalize(
    mean=[-0.3444/0.2034, -0.3803/0.1366, -0.4078/0.1153],
    std=[1/0.2034, 1/0.1366, 1/0.1153]
)

def predict(img):
    tensor = transform(img.convert("RGB")).unsqueeze(0)
    with torch.no_grad():
        probs = torch.softmax(model(tensor), dim=1)[0].numpy()
    pred_idx = probs.argmax()
    
    cam = GradCAM(model=model, target_layers=[model.conv_head])
    gcam = cam(input_tensor=tensor, 
               targets=[ClassifierOutputTarget(pred_idx)])[0]
    rgb = inv_norm(tensor[0]).permute(1,2,0).numpy().clip(0,1)
    overlay = show_cam_on_image(rgb.astype(np.float32), gcam, use_rgb=True)
    
    scores = {CLASS_NAMES[i]: float(probs[i]) for i in range(len(CLASS_NAMES))}
    return Image.fromarray(overlay), scores

demo = gr.Interface(
    fn=predict,
    inputs=gr.Image(type="pil", label="Upload Satellite Image"),
    outputs=[
        gr.Image(label="Grad-CAM Heatmap"),
        gr.Label(label="Class Probabilities", num_top_classes=10)
    ],
    title="🛰️ Satellite Land Use Classifier",
    description="EfficientNet-B0 trained on EuroSAT · 95.7% accuracy · 10 land-use classes",
    examples=[]
)

demo.launch()