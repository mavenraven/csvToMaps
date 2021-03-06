package org.mavenraven;

import com.mapbox.geojson.LineString;
import java.time.Duration;

public class Walk {
    private final LineString lineString;
    private final double distanceTraveled;
    private final Duration totalTime;

    public Walk(
            LineString lineString,
            double distanceTraveled,
            Duration totalTime) {
        this.lineString = lineString;
        this.distanceTraveled = distanceTraveled;
        this.totalTime = totalTime;
    }

    public LineString getLineString() {
        return lineString;
    }

    public double getDistanceTraveledInMeters() {
        return distanceTraveled;
    }

    public Duration getTotalTime() {
        return totalTime;
    }
}
