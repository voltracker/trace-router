import Globe from "react-globe.gl";
import React, { useState, useEffect } from "react";
import "./App.css";

function App() {
  const nodesUrl = "http://localhost:8080/api/v1/nodes/";
  const aggsUrl = "http://localhost:8080/api/v1/aggs/";
  const [nodes, setNodes] = useState(null);
  const [aggs, setAggs] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch(nodesUrl);
        const aggsResponse = await fetch(aggsUrl);
        if (!response.ok) {
          throw new Error(`Response status: ${response.status}`);
        }
        if (!aggsResponse.ok) {
          throw new Error(
            `Aggregations response status: ${aggsResponse.status}`,
          );
        }
        const result = await response.json();
        var max = 0;
        for (var x = 0; x < result.length; x++) {
          if (result[x].count > max) {
            max = result[x].count;
          }
        }
        const toUse = result.map(({ ip, count, lat, lon }) => ({
          ip: ip,
          size: count / max / 2,
          lat: lat,
          lon: lon,
        }));

        const refmap = {};
        result.forEach(({ ip, lat, lon }) => {
          refmap[ip] = { lat, lon };
        });

        const aggResult = await aggsResponse.json();
        console.log(aggResult);
        const arcsData = aggResult
          .filter(
            ({ Source_ip, Dest_ip }) => refmap[Source_ip] && refmap[Dest_ip],
          )
          .filter(
            ({ Source_ip, Dest_ip }) =>
              Source_ip != "localhost" && Dest_ip != "localhost",
          )
          .map(({ Source_ip, Dest_ip, Count, Avg_latency }) => ({
            src_lat: refmap[Source_ip].lat,
            src_lon: refmap[Source_ip].lon,
            dst_lat: refmap[Dest_ip].lat,
            dst_lon: refmap[Dest_ip].lon,
            count: Count,
            label:
              Source_ip +
              " -> " +
              Dest_ip +
              "<br>" +
              "latency: " +
              Avg_latency +
              "ms",
            latency: Avg_latency * 30,
          }));
        setAggs(arcsData);
        setNodes(toUse);
        console.log(result);
      } catch (error) {
        console.error(error.message);
      }
    };
    fetchData();
  }, []);

  return (
    <>
      <h1>trace-router</h1>
      {nodes && aggs && (
        <Globe
          pointsData={nodes}
          pointLng="lon"
          pointLat="lat"
          pointLabel="ip"
          pointRadius={0.02}
          pointAltitude="size"
          pointColor={() => "#FFF000"}
          arcsData={aggs}
          arcLabel="label"
          arcDashAnimateTime="latency"
          arcsTransitionDuration={100}
          arcDashGap={0.75}
          arcColor={(d) => [`rgba(0, 255, 0, 0.4)`, `rgba(255, 0, 0, 0.4)`]}
          arcStartLat="src_lat"
          arcStartLng="src_lon"
          arcEndLat="dst_lat"
          arcEndLng="dst_lon"
          globeImageUrl="https://cdn.jsdelivr.net/npm/three-globe/example/img/earth-blue-marble.jpg"
          bumpImageUrl="https://cdn.jsdelivr.net/npm/three-globe/example/img/earth-topology.png"
        />
      )}
    </>
  );
}

export default App;
