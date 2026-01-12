import pandas as pd
import joblib
import numpy as np
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler
from sklearn.ensemble import RandomForestClassifier

print("🔄 Chargement et nettoyage des données...")

# 1. Chargement
try:
    df = pd.read_csv('Algerian_forest_fires_dataset_CLEANED.csv')
except FileNotFoundError:
    print("❌ ERREUR : Fichier CSV introuvable.")
    exit(1)

# 2. Nettoyage des Noms de Colonnes (Le correctif pour "Classes  ")
# On enlève les espaces avant/après chaque nom de colonne
df.columns = df.columns.str.strip()
print(f"✅ Colonnes nettoyées : {df.columns.tolist()}")

# 3. Nettoyage des Données (Le correctif pour "7.1 ")
features = ['Temperature', 'RH', 'Ws', 'Rain', 'FFMC', 'DMC', 'DC', 'ISI', 'BUI', 'FWI']

# On force la conversion de toutes les features en numérique
# 'coerce' va transformer les valeurs illisibles en NaN (vide) au lieu de planter
for col in features:
    if col in df.columns:
        df[col] = pd.to_numeric(df[col], errors='coerce')

# On supprime les lignes qui contiendraient maintenant des vides (NaN)
df = df.dropna(subset=features)

# 4. Préparation de la cible (Classes)
if 'Classes' not in df.columns:
    print("❌ ERREUR : Colonne 'Classes' introuvable.")
    exit(1)

# On nettoie la colonne Classes (enlève les espaces et met en minuscule)
df['Classes'] = df['Classes'].astype(str).str.strip().str.lower()
# On convertit : si le mot 'fire' est présent -> 1, sinon -> 0
df['target'] = df['Classes'].apply(lambda x: 1 if 'fire' in x and 'not' not in x else 0)

# Vérification rapide
print(f"📊 Données prêtes : {len(df)} lignes valides.")
print(f"   Répartition : {df['target'].value_counts().to_dict()} (1=Feu, 0=Pas feu)")

# 5. Entraînement
print("🚀 Entraînement du modèle...")
X = df[features]
y = df['target']

scaler = StandardScaler()
X_scaled = scaler.fit_transform(X)

X_train, X_test, y_train, y_test = train_test_split(X_scaled, y, test_size=0.3, random_state=42)

model = RandomForestClassifier(n_estimators=100, random_state=42)
model.fit(X_train, y_train)

# Score rapide pour vérifier que ça marche
score = model.score(X_test, y_test)
print(f"🎯 Précision du modèle : {score:.2f}")

# 6. Sauvegarde
joblib.dump(model, 'rf_fire_model.pkl')
joblib.dump(scaler, 'scaler.pkl')

print("💾 SUCCÈS : 'rf_fire_model.pkl' et 'scaler.pkl' ont été créés !")
